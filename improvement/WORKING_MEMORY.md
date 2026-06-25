# Working Memory

The improvement loop's persistent state across runs. The loop reads this at the start of every
run and updates it at the end. Keep it concise: append iteration entries, keep ONE current
baseline block, and prune notes older than ~30 days unless still relevant. This file is committed
to the repo via each PR (or directly when a run only grooms/records).

> Tip: keep this file small. If it grows large, move resolved history to `improvement/ARCHIVE.md`.

## Current baseline
_Last measured: 2026-06-25 on commit 038fee7 (branch master, after PR #2 merged)_

| metric        | value                          | how measured                          |
|---------------|--------------------------------|---------------------------------------|
| tests         | PASS (all pkgs green)          | `go test ./...` (pure-Go path)        |
| coverage      | not tracked                    | —                                     |
| benchmark(s)  | none                           | (no benchmark suite configured)       |
| go vet        | clean (exit 0)                 | `go vet ./...`                        |
| ui lint       | **RED at baseline** (2 errors) | `pnpm -C ui run lint` — see ticket #5 |
| ui build      | PASS                           | `pnpm -C ui run build`                |

Note: the `lint` gate is `go vet ./... && pnpm -C ui run lint`. `go vet` is clean, but
`pnpm -C ui run lint` already fails on `master` with 2 pre-existing eslint errors in
`ui/src/components/PrivateRoute.tsx` and `ui/src/pages/Integrations/index.tsx` (filed as #5).
Treat UI-lint as a known-red baseline until #5 lands; Go-only changes are unaffected.

Note: `go test ./...` requires `ui/dist` to exist (the `ui/assets.go` `//go:embed dist/*`).
Build it first with `pnpm -C ui install --frozen-lockfile && pnpm -C ui run build`, otherwise the
packages that import the embed fail at setup with "pattern dist/*: no matching files found".
This is a build-ordering artifact, not a code regression.

## Budget tally (current month)
- Month: 2026-06
- Increments merged: 1 (PR #2 merged → master @038fee7); 1 PR open (#4, awaiting human review)
- Approx. tokens used: n/a (monthly_token_budget = 0, no cap)

## Metric trend (for diminishing-returns detection)
_Most recent increments and their effect on the targeted metric._
- 2026-06-24 — axis: security/robustness — Δ: closed 2 missing-timeout call sites + 1 response-body
  leak; added webhook test coverage (suite stays green) — PR: #1 branch auto-improve/1-http-timeouts
  (merged as PR #2 → master)
- 2026-06-25 — axis: security/robustness — Δ: bounded 4 unbounded `io.ReadAll(c.Request.Body)`
  control endpoints with `http.MaxBytesReader` (DoS guard); +2 tests; suite stays green — PR #4
  branch auto-improve/3-bound-request-bodies. (2 increments on this axis; window=5, no diminishing
  returns yet — but consider switching axes if the next one also lands here.)

## Failed / rejected approaches (do not blindly retry)
_Record what was tried and why it failed so the loop doesn't loop._
- (none yet)

## Decisions & notes
_Durable choices worth remembering (e.g. "library X chosen over Y because …")._
- 2026-06-24 — Convention: outbound `http.Client`s carry a 30s timeout (originated in
  `internal/integrations/ics.go:98`). New outbound calls should match this.
- 2026-06-24 — Default branch is `master` (not `main`); treat `master` as the protected branch.

## Iteration log
_One line per run. Newest at top._
- 2026-06-25 — phase: auto — ticket #3 — outcome: PR #4 opened (bound 4 unbounded request-body reads
  with http.MaxBytesReader + 2 tests) → master. Empty backlog at start; also groomed #5 (red UI lint
  baseline). PR: auto-improve/3-bound-request-bodies.
- 2026-06-24 — phase: auto — ticket #1 — outcome: PR opened (HTTP timeouts + HWR body close + webhook
  test) — PR: auto-improve/1-http-timeouts → master (merged as #2)
