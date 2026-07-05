# Colosseum

**An adversarial evaluation engine for coding agents — presented as a live AI battleground.**

LLMs fight each other at competitive programming on a sandboxed judge. In the
**Race** format they solve the same problem head-to-head; in **Attack/Defense**
one model writes a solution and the other writes a test input designed to break
it, validated against a pinned reference solution. Every match is an append-only
event log you can replay, and a tournament ladder ranks models with Elo +
confidence intervals across the axes standard benchmarks ignore.

```
$ colosseum match --problem max-subarray --a mock:wrong --b mock:reference --format ad
⚔  Maximum Subarray Sum  ad · mock:wrong vs mock:reference
  ▸ attack: B → A
  ⚑ B attacks with input 9⏎-2 1 -3 4 -1 2 1 -5 4
  ⚔ B → BROKE DEFENDER  defender output disagrees with the reference
  ▸ attack: A → B
  ⚔ A → shielded  defender matched the reference
WINNER: B  (broke_and_survived)
```

---

## Why this exists (the argument a benchmark can't make)

On easy problems, **pass@1 saturates** — every capable model scores 100%, and the
standard leaderboard says they're equal. They aren't. Colosseum measures the
axes a pass-rate benchmark throws away:

- **Adversarial robustness** — can another model find an input, valid under the
  constraints, that your "100%-correct" solution gets wrong? (Attack/Defense Elo)
- **Efficiency** — tokens and wall-clock time to reach a green solution. (Race)

**The headline result:** models that *tie on pass rate* get *separated by
adversarial robustness*. That divergence is the finding, and the ladder reports
it explicitly:

```
solve-rate ranking:  [model-x  model-y]     # tied — pass rate can't order them
robustness ranking:  [model-y  model-x]     # attack/defense can
→ DIVERGENCE: models that tie on pass rate are separated by adversarial robustness.
```

**A real run** (two local Ollama models, `docs/eval-report-sample.txt`) shows the
honest flip side — and that the tool doesn't invent findings:

```
model                   games solve%   avgTok  survive%    break%  robustElo
qwen2.5-coder:1.5b          4   100%      434      100%        0%     1531±0
llama3.2:latest             4    50%      310      100%        0%     1469±0
→ no divergence yet (add models/problems, or the field is already stratified by pass rate).
```

Here the models *don't* tie — qwen solves everything, llama half — so the pass
rate already separates them and Colosseum correctly reports **no** divergence.
Divergence surfaces when you feed it models of comparable pass rate; that's the
regime the eval is built to illuminate, and the mechanism is covered by a unit
test (`rank.TestReportDetectsDivergence`) and the oracle-validated break demo above.

It's also a spectacle you can watch — and that's deliberate: the matches *are*
the content, so the project needs zero user base to be interesting. A recruiter
clicks a replay and sees a finished brawl in ten seconds, then scrolls into the
methodology.

## What makes it not the generic "LLM + wrapper" project

| Layer | What's actually hard | Where to look |
|---|---|---|
| **Sandbox** | Auto-executing code written by *adversarially-prompted* LLMs. Fork bombs, OOM, network exfiltration, fs escape — all contained, all tested in CI. | [`docs/SANDBOX.md`](docs/SANDBOX.md), `internal/judge` |
| **Adversarial eval** | An attack only counts if the reference oracle disagrees with the defender on an input the reference itself handles. No judging by vibes. | [`docs/METHODOLOGY.md`](docs/METHODOLOGY.md), `internal/match/attackdefense.go` |
| **Event sourcing** | One append-only log per match powers live spectating *and* replay *and* the eval's raw data — one source of truth, three consumers. | [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), `internal/events` |
| **Ranking** | Elo with bootstrap confidence intervals, because a ranking from 8 games is noise and the CI shows it. | `internal/rank` |

## Architecture

```
                 ┌──────────────────────────────────────────┐
                 │  Match  (state machine + Format plugin)    │
                 │            Race │ AttackDefense            │
                 └───────┬───────────────────────┬────────────┘
   Fighter A ───────────►│  solve→judge→debug     │◄────────── Fighter B
   (provider:            │  loop, token budgets   │           (Anthropic /
    anthropic/ollama/    └───────────┬────────────┘            Ollama / mock)
    openai/mock)                     │ submissions
                                     ▼
                          Judge  (Docker sandbox)
                    per-run container · --network none
                    memory/pids/cpu limits · read-only rootfs
                    non-root · wall-clock watchdog
                                     │ per-case results
                                     ▼
                    Event log (append-only, per match)
                    ┌──────────────┬─────────────────┐
                    ▼              ▼                 ▼
               live tail       replay          eval export
             (reconnect-      (timed         (→ rank: Elo,
              and-replay)     re-render)      CIs, divergence)
```

A `Format` is a plugin (`internal/match`): Race and Attack/Defense implement the
same interface, so adding a format is new logic, not new plumbing. Every match
carries a **run manifest** (models, prompt persona, problem version, judge image,
seed, budgets) so results are reproducible and replays are auditable.

## Quickstart

Requires **Go 1.26+** and a **Docker** engine (Docker Desktop, OrbStack, or
colima). No API key needed for the offline demo — `mock:` fighters replay pinned
solutions and edge-case attacks.

```bash
# 1. Judge a single solution against a problem's hidden tests (in the sandbox)
colosseum judge --problem sum-two --file mysolution.py

# 2. Run an AI-vs-AI match, fully offline, and save a replayable record
colosseum match --problem max-subarray --a mock:wrong --b mock:reference --format ad

# 3. Replay it from the event log (the same stream the live view renders)
colosseum replay --file data/matches/<id>.json          # timed
colosseum replay --file data/matches/<id>.json --jsonl  # structured export

# 4. Run a tournament and print the eval report
colosseum ladder --fighters mock:reference,mock:wrong --formats race,ad --rounds 2

# 5. Watch replays in the browser (single static binary, no build step)
colosseum serve   # → http://localhost:8080
```

The spectator UI (`internal/web`, a `go:embed`'d single-page app — no npm, no
build) lists every saved match and replays it in-browser from the same event
log the terminal renders, with a leaderboard fed by the ladder's report.

### Running with real models

```bash
export ANTHROPIC_API_KEY=...
colosseum ladder --fighters anthropic:claude-haiku-4-5,anthropic:claude-sonnet-5 \
  --formats race,ad --rounds 3

# Or free & local via Ollama (OpenAI-compatible endpoint):
ollama serve && ollama pull qwen2.5-coder
colosseum ladder --fighters ollama:qwen2.5-coder,ollama:deepseek-coder-v2 --formats ad
```

Fighter specs: `anthropic:<model>` · `ollama:<model>` · `openai:<model>` ·
`mock:reference` · `mock:wrong`.

## The sandbox, tested — not claimed

The containment story is enforced by CI, running real hostile programs against
real containers (`internal/judge/sandbox_docker_test.go`):

| Attack | Control | Asserted outcome |
|---|---|---|
| `while True: os.fork()` | `--pids-limit` | contained, non-AC verdict, returns fast |
| `while True: pass` | wall-clock watchdog | `TLE` |
| unbounded allocation | `--memory` (swap off) | `MLE` (OOM kill) |
| `socket.connect(1.1.1.1:53)` | `--network none` | never reaches the network |
| `open('/pwned','w')` | `--read-only` rootfs | `RE`, no write |

Full threat model and the gVisor/Firecracker upgrade path: [`docs/SANDBOX.md`](docs/SANDBOX.md).

## Testing

```bash
go test ./...            # everything (Docker tests included)
go test -short ./...     # skip the Docker integration + security suite
```

Coverage spans the judge security suite (real containers), the verdict state
machine, both format plugins, the Elo/CI math, event-log reconnect-replay, and
end-to-end Race and Attack/Defense matches driven by scripted models against the
real judge (zero API cost).

## Honest limitations

- **Shared-kernel isolation.** Containers are namespaces, not VMs; the `Runner`
  interface is the seam to swap in gVisor/Firecracker for a hostile-internet
  deployment. Documented, not hand-waved.
- **Divergence needs comparable models.** With one strong and one weak fighter,
  the strong one dominates both axes and nothing diverges — that's honest. The
  interesting result appears with models of *similar* pass rate; that's what the
  eval is built to surface.
- **Toy problems by design.** Real-repo tasks were explicitly out of scope; the
  novelty is in the adversarial *measurement*, not problem difficulty.
- **Attacker sees only pass-count feedback** during the debug loop, never the
  hidden expected outputs — mirroring real judges ("Wrong Answer on test N").

## Layout

```
cmd/colosseum        # CLI: judge | match | replay | ladder | serve
internal/judge       # Docker sandbox + CI security suite  ← the crown jewel
internal/match       # state machine + Race + AttackDefense (oracle-validated)
internal/agent       # provider-agnostic fighters (anthropic/ollama/openai/mock)
internal/events      # append-only log, reconnect-replay, JSONL export
internal/rank        # Elo + bootstrap CIs + divergence report
internal/ladder      # tournament runner
internal/web         # go:embed'd spectator UI (browser replays + leaderboard)
problems/            # versioned: statement, hidden cases, constraints, reference
docs/                # SANDBOX.md · METHODOLOGY.md · ARCHITECTURE.md
```
