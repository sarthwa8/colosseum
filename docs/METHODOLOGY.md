# Evaluation Methodology

Colosseum is an eval engine wearing a battleground's clothes. This document is
the "why the numbers mean something" half — the scoring axes, how an attack is
validated, and what reproducibility guarantees hold.

## The problem with pass@1 on easy tasks

On toy competitive-programming problems, capable models all reach 100% pass rate.
A pass-rate leaderboard therefore reports a tie and stops. But two models that
both "solve" a problem are not equally good: one may have written a solution that
is correct only on the test cases it was shown, and wrong on an adjacent input a
determined adversary can find. Pass@1 is blind to that. Colosseum is built to see
it.

## Three scoring axes

| Axis | Format | What it captures | Why pass@1 misses it |
|---|---|---|---|
| **Correctness** | Race, A/D | Does the solution pass the hidden tests? | This *is* pass@1 — and it saturates. |
| **Robustness** | Attack/Defense | Does the solution survive an *adversarially chosen* input? | The adversary probes beyond the fixed test set. |
| **Efficiency** | Race | Tokens and wall-clock to a green solution. | Pass@1 ignores cost entirely. |

The **divergence between the correctness ranking and the robustness ranking** is
the headline result. When models tie on correctness but the robustness Elo orders
them, the standard benchmark cannot distinguish models that Colosseum can.

## Attack validation — the reference oracle

An attack in the Attack/Defense format is a proposed standard-input string. It
would be trivial (and meaningless) to "count" any input on which the defender
produces surprising output — an out-of-spec input can make *any* program
misbehave. So an attack counts as a **break** only if **both** hold:

1. **The reference solution handles the input cleanly** — it runs to a normal
   exit within the resource limits. If the pinned reference can't process the
   input, the input is out-of-spec and the attack is *rejected as invalid*.
2. **The defender disagrees with the reference** — the defender either crashes /
   times out / blows a limit, or produces output that differs from the
   reference's (compared under the same whitespace-lenient rule the judge uses
   for verdicts).

This makes the pinned reference solution the oracle and the constraints its
implicit contract. `internal/match/attackdefense.go::validateAttack` implements
exactly this; the end-to-end test seeds a classic bug (max-subarray with the
running best initialized to `0`, wrong for all-negative arrays) and asserts the
correct defender breaks the buggy one while surviving the reverse attack.

## Winner rules

- **Race** — first to pass all hidden tests wins (fastest wall-clock among
  solvers). If neither solves within the debug-iteration limit, the tiebreak is
  more cases passed, then fewer tokens — the efficiency axis.
- **Attack/Defense** — a fighter that broke its opponent *and* survived the
  opponent's attack wins. Symmetric results (both broke, or both survived) fall
  back to who actually solved the problem, then fewer tokens, then a draw.

Budget exhaustion and provider errors are recorded honestly as **forfeits**, not
silently dropped — a model that ran out of its token budget forfeits the round.

## Ratings: Elo with confidence intervals

Per-format Elo (K=32, base 1500) is computed over the match results. Crucially,
each rating ships with a **95% bootstrap confidence interval** (`internal/rank`):
the result set is resampled with replacement N times, Elo is recomputed each
time, and the 2.5/97.5 percentiles form the interval. With few games the interval
is wide — which is the point. A ranking from a handful of matches is noise, and
the CI makes that visible instead of pretending a 12-point Elo gap over 8 games
is real.

## Reproducibility

Every match stores a **run manifest**: each fighter's provider/model/prompt
persona, the problem slug **and version**, the max-iteration and token budgets,
a seed, and the creation time. Problems are versioned (`problem.json`), so a
change to a statement or test set bumps the version and old results remain
attributable to the inputs that produced them. The full event log is persisted
alongside, so any match can be replayed or audited byte-for-byte after the fact,
and the ladder's schedule + bootstrap are seeded for repeatable runs.

## Known threats to validity (stated, not hidden)

- **Reference solution is trusted.** If the reference is itself buggy, a "break"
  might reflect the reference's error, not the defender's. Mitigation: references
  are hand-verified against the hidden cases before a problem ships.
- **Attacker feedback.** During the solve/debug loop a fighter sees only its
  verdict and pass count (plus its own stderr) — never the hidden expected
  outputs — so the loop can't overfit to leaked tests.
- **Small samples.** Elo over a short ladder is high-variance; the CIs are the
  guardrail, and the honest reading is "run more matches before trusting a gap."
- **Prompt sensitivity.** Fighter personas are fixed and versioned in the
  manifest; changing them changes the eval, which is why they're recorded.
