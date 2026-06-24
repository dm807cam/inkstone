---
name: eval-and-benchmark
description: >-
  The measurement backbone for the improvement loop: establish a baseline, run the regression
  and benchmark suite, compare before vs after, and detect reward hacking and diminishing
  returns via holdout checks. Use to decide whether ANY change is a real improvement before it
  becomes a PR, and whenever asked to measure, benchmark, or validate an increment.
---

# Eval and Benchmark (measurement keystone)

Nothing else in the loop is trustworthy without this skill: it is how "change" is told apart from
"improvement". Every increment passes through here before it can become a PR.

## 1. Establish the baseline
From `improvement/config.yml`, run the canonical commands for the project:
- **Correctness:** the regression/test suite (record pass/fail counts and coverage if available).
- **Performance:** the benchmark command(s) (record the key numbers — latency, throughput,
  memory, or a domain metric like accuracy/F1). Run benchmarks with enough repetition to be
  meaningful; note variance, not just the mean. Pin inputs and seeds where possible.
- **Quality/security gates:** lint, type-check, complexity, and the security scan.
Snapshot these as the **before** numbers and reconcile them against the last baseline stored in
`WORKING_MEMORY.md`. An unexplained drift from the stored baseline is itself a finding.

## 2. Measure after
Re-run the *same* commands on the change. Present a compact **before → after** table for every
gate metric. A change is only a candidate if the targeted metric improved (or, for a pure
refactor, held flat) and **no other gate regressed**.

## 3. Reward-hacking / Goodhart checks (mandatory)
An optimizer will exploit the metric if it can. Before accepting any change, confirm none of these:
- A test, assertion, fixture, golden file, or benchmark input was **deleted, weakened, skipped,
  or hardcoded** to pass. Diff the test/benchmark files specifically and call out every change.
- The benchmark now measures something easier (smaller/cached input, removed warmup, fewer iters).
- The gain comes from special-casing the eval inputs rather than the general problem.
- **Holdout:** keep a holdout set / secondary benchmark (configured in `config.yml`) that the
  change was *not* written against, and confirm the improvement holds there too. If a gain
  appears on the primary metric but not the holdout, treat it as overfitting and reject.
Any of the above ⇒ the change **fails the gate**, regardless of the headline number.

## 4. Diminishing returns
Append each accepted increment's metric delta to the trend in `WORKING_MEMORY.md`. If the last
`N` increments (N from `config.yml`) produced negligible movement on an axis, signal the
orchestrator to raise the acceptance bar or switch axes — don't keep churning for noise-level wins.

## 5. Drift / periodic re-validation
On the cadence in `config.yml` (e.g. weekly), re-validate against ground truth rather than the
previous step: re-run the full suite from a clean checkout and reconcile with the stored baseline,
so small errors compounding over many autonomous runs are caught early.

## Output contract
Emit a structured verdict the orchestrator can act on:
`PASS` (with the before→after table) only when the target metric improved for a legitimate reason
and every other gate held; otherwise `FAIL` with the specific reason (regression, weakened
measurement, holdout miss, or no meaningful movement). Never report a metric without saying how it
was measured.
