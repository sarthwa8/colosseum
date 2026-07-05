# Architecture

Two design decisions carry most of the weight: **event sourcing** (one append-only
log per match is the single source of truth) and the **`Format` plugin interface**
(contest types are data-driven, not branches). Everything else follows.

## Packages and their dependency direction

```
cmd/colosseum ──► ladder ──► match ──► agent
                    │          │  │     judge
                    │          │  └───► events
                    └──► rank ─┘        problem
```

- `judge` — the Docker sandbox. Depends on nothing internal. Exposes a `Runner`
  interface so the container backend is swappable (Docker today; gVisor/Firecracker
  is a drop-in).
- `agent` — provider-agnostic LLM fighters (Anthropic SDK, an OpenAI-compatible
  HTTP client for Ollama, and mocks). Knows nothing about matches.
- `events` — the append-only log with live pub/sub and a replay cursor. Pure data.
- `match` — the state machine, the `Format` interface, and the two formats. Wires
  agent + judge + events together. **Does not import `rank`.**
- `rank` — Elo, bootstrap CIs, and the eval report. Imports `match` (for the
  `Outcome`/`FighterScore` types it aggregates). One-directional; no cycle.
- `ladder` — sits on top of `match` + `rank`, which is *why* those two don't
  import each other: the tournament runner is the only place they meet.

## Event sourcing: one log, three consumers

A match doesn't hold "state" that gets mutated and rendered. It **emits events**
to an append-only log (`internal/events`). Each event has a per-match monotonic
sequence number, an actor, a timestamp, and a typed payload.

```
Match.Run ──emits──► Event log ──┬──► live tail   (WebSocket-ready pub/sub)
                                 ├──► replay       (events.Replay re-times the stream)
                                 └──► eval export  (JSONL / rank.Build)
```

Because live spectating and replay render the **identical** event stream, there's
one rendering contract, not two. And the log *is* the eval's raw data — no
separate metrics pipeline that can drift from what actually happened. `rank.Build`
consumes match outcomes derived from the same source.

### Reconnect-and-replay

`Log.Subscribe(fromSeq)` first replays every stored event with `Seq >= fromSeq`,
then delivers live events. That single primitive gives a client that dropped at
sequence N a gap-free, dupe-free resume from N+1 — the reconnection story a flaky
connection needs, tested in `internal/events/events_test.go`.

## The Format plugin

```go
type Format interface {
    Name() string
    Run(ctx context.Context, m *Match) (Outcome, error)
}
```

`Race` and `AttackDefense` implement it. The `Match` gives a format everything it
needs (fighters, judge, problem, event log, budgets); the format decides the
phases and the scoring. Adding `debug-duel` or `code-golf` is a new file
implementing `Run`, not a new `switch` threaded through the engine.

The shared solve→judge→debug loop (`match/solve.go`) is factored out so both
formats reuse it: Race races two solve loops; Attack/Defense runs a solve loop to
produce each defender's solution, then a single adversarial turn to attack it.

## The judge's control flow (why it's careful)

Running untrusted code correctly is mostly about *not trusting it to terminate or
report honestly*:

- **External watchdog, not `context` cancellation.** Cancelling the Go context
  would kill the `docker` client and orphan the container. Instead each container
  is named, and a wall-clock timer issues an independent `docker kill`. The
  container is force-removed on every exit path.
- **Verdict from `docker inspect`, not the exit code.** A killed process exits
  137 whether the kernel OOM-killed it or the watchdog did. The judge reads
  `State.OOMKilled` and `State.ExitCode` after the run to tell `MLE` from `TLE`.
- **Non-blocking capped output.** A capped writer always fully consumes the pipe
  (returning `len(p)`) so a program spewing gigabytes can't deadlock the reader;
  bytes past the cap are dropped and the output is flagged truncated.

## Concurrency model

- The **judge** runs test cases through a bounded worker pool of containers.
- **Race** runs both fighters' solve loops as goroutines and cancels the loser
  the instant anyone reaches Accepted (saving tokens and wall time).
- The **ladder** runs matches *sequentially* on purpose — the judge is already
  internally concurrent, and serializing matches keeps Docker and provider rate
  limits sane.
- The **event log** fans out to subscribers over buffered channels; a slow
  subscriber is dropped rather than allowed to stall the match (it can re-sync via
  `Subscribe`).

## Extension points

| Want to… | Touch |
|---|---|
| Add a language (C++, Go) | `internal/judge/language.go` — it's a config entry |
| Add a contest format | a new `Format` in `internal/match` |
| Add a model provider | implement `agent.Provider` |
| Harden isolation | swap `DockerRunner` for a gVisor/Firecracker `Runner` |
| Add a problem | a directory under `problems/` (statement, cases, constraints, reference) |
