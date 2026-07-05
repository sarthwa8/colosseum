# Sandbox Threat Model

Colosseum executes code written by adversarially-prompted LLMs. In the
attack/defense format we *explicitly ask* one model to break another's program,
so hostile code isn't an edge case — it's the workload. This document states
what the judge defends against, how, and what it deliberately does not.

## Trust boundary

Everything inside a judged container is **untrusted**. The host, the Docker
daemon, other containers, the network, and the filesystem outside the mounted
work directory are all **outside** the trust boundary. The judge's job is to run
an untrusted program, capture its stdout/exit status, and guarantee it cannot
affect anything else — then throw the container away.

## Adversary capabilities we assume

The submitted program may attempt to:

1. Run forever (denial of service via CPU).
2. Spawn unbounded processes/threads (fork bomb).
3. Allocate unbounded memory.
4. Produce unbounded output (fill disk / exhaust the reader).
5. Reach the network (exfiltrate data, phone home, attack other hosts).
6. Read or write the host filesystem.
7. Escalate privileges (setuid binaries, capabilities).
8. Refuse to terminate to keep resources pinned.

## Controls

Every judged run is a fresh, disposable container created with these flags
(`internal/judge/sandbox.go`, `securityArgs`). Each maps to a capability above.

| Control | Flag | Defeats |
|---|---|---|
| No network | `--network none` | (5) egress / C2 / lateral movement |
| Memory ceiling | `--memory Nm --memory-swap Nm` | (3) memory exhaustion — equal swap disables swap so the OOM killer fires deterministically |
| CPU quota | `--cpus C` | (1) CPU monopolization |
| Process cap | `--pids-limit P` | (2) fork bombs |
| Read-only root | `--read-only` + `--tmpfs /tmp:...,noexec` | (6) host/root writes; `noexec` blocks executing dropped binaries from scratch |
| Drop capabilities | `--cap-drop ALL` | (7) raw sockets, mounts, ptrace, etc. |
| No privilege gain | `--security-opt no-new-privileges` | (7) setuid escalation |
| Non-root user | `--user 65534:65534` (nobody) | (6)(7) blast radius if any control is bypassed |
| Isolated workdir | `-v <tmp>:/sandbox:ro` (rw only during compile) | (6) code sees only its own throwaway dir, never host files |
| Wall-clock watchdog | independent timer → `docker kill -s KILL` | (1)(8) guarantees termination regardless of the program |
| Output cap | capped reader, always drains the pipe | (4) bounds captured output without deadlocking |

### Why an external watchdog

Cancelling the Go `context` would kill the `docker` *client* and orphan the
*container*. Instead the judge names each container and, on timeout, issues an
independent `docker kill --signal=KILL`, then force-removes it on every exit
path. Termination never depends on the untrusted process cooperating.

### Why `docker inspect` for the verdict

A killed process exits 137 whether it was OOM-killed or watchdog-killed. The
judge reads `State.OOMKilled` and `State.ExitCode` after the run to distinguish
**MLE** (kernel OOM) from **TLE** (our watchdog) — the exit code alone is
ambiguous.

## Verified in CI

`internal/judge/sandbox_docker_test.go` runs real hostile programs against real
containers and asserts containment. These are part of the test suite, not prose:

| Test | Program | Asserted outcome |
|---|---|---|
| `ForkBombContained` | `while True: os.fork()` | returns bounded, never `AC` (pids-limit) |
| `InfiniteLoopTLE` | `while True: pass` | `TLE` (watchdog) |
| `MemoryBombMLE` | append 20 MiB chunks forever | `MLE` (OOM killer) |
| `NetworkBlocked` | `socket.connect(1.1.1.1:53)` | never reaches network, non-`AC` |
| `ReadOnlyFilesystem` | `open('/pwned','w')` | `RE`, no write |
| `HappyPath` | correct solution | `AC` — controls don't break normal code |

## Known limitations (honest)

- **Shared kernel.** Containers are namespaces, not VMs. A kernel 0-day could
  escape. For a hostile-internet deployment the `Runner` interface is the seam:
  swap `DockerRunner` for **gVisor** (`runsc`), **Kata**, or **Firecracker**
  microVMs with zero changes above it. This is the documented upgrade path.
- **Compile phase runs the compiler untrusted** with a writable mount. It is
  contained by every other control (no network, pids/mem limits, non-root), but
  it does execute attacker-influenced toolchain input.
- **Wall-clock, not CPU-time, limits.** A program throttled by `--cpus` is
  measured in real time; the wall limit is set with that headroom in mind.
- **No seccomp profile beyond Docker's default.** A custom syscall allowlist
  would tighten (7) further and is a reasonable next step.

The design principle throughout: **fail closed** — every stage that can't prove
safety kills the container rather than trusting it.
