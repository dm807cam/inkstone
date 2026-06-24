# Working Memory

The improvement loop's persistent state across runs. The loop reads this at the start of every
run and updates it at the end. Keep it concise: append iteration entries, keep ONE current
baseline block, and prune notes older than ~30 days unless still relevant. This file is committed
to the repo via each PR (or directly when a run only grooms/records).

> Tip: keep this file small. If it grows large, move resolved history to `improvement/ARCHIVE.md`.

## Current baseline
_Last measured: 2026-06-24 on commit c82d702 (branch master)_

| metric        | value                          | how measured                          |
|---------------|--------------------------------|---------------------------------------|
| tests         | PASS (all pkgs green)          | `go test ./...` (pure-Go path)        |
| coverage      | not tracked                    | —                                     |
| benchmark(s)  | none                           | (no benchmark suite configured)       |
| lint/security | `go vet ./...` clean (exit 0)  | `go vet ./...`                        |

Note: `go test ./...` requires `ui/dist` to exist (the `ui/assets.go` `//go:embed dist/*`).
Build it first with `pnpm -C ui install --frozen-lockfile && pnpm -C ui run build`, otherwise the
packages that import the embed fail at setup with "pattern dist/*: no matching files found".
This is a build-ordering artifact, not a code regression.

## Budget tally (current month)
- Month: 2026-06
- Increments merged: 0 (1 PR open, awaiting human review)
- Approx. tokens used: n/a (monthly_token_budget = 0, no cap)

## Metric trend (for diminishing-returns detection)
_Most recent increments and their effect on the targeted metric._
- 2026-06-24 — axis: security/robustness — Δ: closed 2 missing-timeout call sites + 1 response-body
  leak; added webhook test coverage (suite stays green) — PR: #1 branch auto-improve/1-http-timeouts

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
- 2026-06-24 — phase: auto — ticket #1 — outcome: PR opened (HTTP timeouts + HWR body close + webhook
  test) — PR: auto-improve/1-http-timeouts → master
