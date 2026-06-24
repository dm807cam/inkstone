# Working Memory

The improvement loop's persistent state across runs. The loop reads this at the start of every
run and updates it at the end. Keep it concise: append iteration entries, keep ONE current
baseline block, and prune notes older than ~30 days unless still relevant. This file is committed
to the repo via each PR (or directly when a run only grooms/records).

> Tip: keep this file small. If it grows large, move resolved history to `improvement/ARCHIVE.md`.

## Current baseline
_Last measured: (date) on commit (sha)_

| metric        | value | how measured            |
|---------------|-------|-------------------------|
| tests         | —     | (test command)          |
| coverage      | —     | —                       |
| benchmark(s)  | —     | (benchmark command)     |
| lint/security | —     | (scan command)          |

## Budget tally (current month)
- Month: —
- Increments merged: 0
- Approx. tokens used: 0 / (monthly_token_budget)

## Metric trend (for diminishing-returns detection)
_Most recent increments and their effect on the targeted metric._
- (date) — axis: — — Δ: — — PR: —

## Failed / rejected approaches (do not blindly retry)
_Record what was tried and why it failed so the loop doesn't loop._
- (date) — ticket #— — approach: — — why rejected: —

## Decisions & notes
_Durable choices worth remembering (e.g. "library X chosen over Y because …")._
- (date) — —

## Iteration log
_One line per run. Newest at top._
- (date) — phase: — — ticket #— — outcome: — — PR: —
