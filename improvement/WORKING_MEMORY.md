# Working Memory

The improvement loop's persistent state across runs. The loop reads this at the start of every
run and updates it at the end. Keep it concise: append iteration entries, keep ONE current
baseline block, and prune notes older than ~30 days unless still relevant. This file is committed
to the repo via each PR (or directly when a run only grooms/records).

> Tip: keep this file small. If it grows large, move resolved history to `improvement/ARCHIVE.md`.

## Current baseline
_Last measured: 2026-06-26 on commit 158a57b (branch master). Drift since 3e05326: PRs #6–#11 merged (UI-lint fix #6, plus the LLM "Convert to text" HWR feature #7–#11)._

| metric        | value                          | how measured                          |
|---------------|--------------------------------|---------------------------------------|
| tests         | PASS (all pkgs green)          | `go test ./...` (pure-Go path)        |
| coverage      | not tracked                    | —                                     |
| benchmark(s)  | none                           | (no benchmark suite configured)       |
| go vet        | clean (exit 0)                 | `go vet ./...`                        |
| ui lint       | GREEN on master (exit 0)       | `pnpm -C ui run lint` (#6 merged)     |
| ui build      | PASS                           | `pnpm -C ui run build`                |
| ui audit      | FAIL — 40 vulns (20 high)      | `pnpm -C ui audit --audit-level high` |

Note: the `lint` gate `go vet ./... && pnpm -C ui run lint` is now GREEN on `master` — PR #6
(#5 fix) merged, so the 2 baseline eslint errors are gone. UI lint is a usable regression signal again.

Note: the `security_scan` gate (`pnpm -C ui audit --audit-level high`) is RED at baseline — 40
advisories (20 high), mostly build-time devDependencies (vite <=6.4.2, ws via mqtt, esbuild, etc.).
Remediation = dependency upgrades, which are HUMAN-GATED per CLAUDE.md rule 4; the loop must NOT
bump deps autonomously. Flag for a human; do not treat as a loop-actionable regression.

Note: `go test ./...` requires `ui/dist` to exist (the `ui/assets.go` `//go:embed dist/*`).
Build it first with `pnpm -C ui install --frozen-lockfile && pnpm -C ui run build`, otherwise the
packages that import the embed fail at setup with "pattern dist/*: no matching files found".
This is a build-ordering artifact, not a code regression.

## Budget tally (current month)
- Month: 2026-06
- Increments merged: 3 (PR #2 → master @038fee7; PR #4 → master @3e05326; PR #6 → master @fae3885)
- PRs open: 1 (#13 — LoadBlob FD-leak fix for #12, awaiting human review)
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
- 2026-06-25 — axis: code cleanliness — Δ: fixed the 2 pre-existing eslint errors that kept the
  UI-lint gate RED at baseline — `PrivateRoute.tsx` `React.FC<any>` → `React.ComponentType`
  (no-explicit-any), `Integrations/index.tsx` dropped unused `e` param (no-unused-vars).
  `pnpm -C ui run lint` now exits 0; build/go vet/go test stay green; no behavior change.
  PR #6 for #5, branch auto-improve/5-ui-lint. (Axis switched away from security/robustness as planned.)
- 2026-06-26 — axis: security/robustness + correctness — Δ: fixed a file-descriptor leak in
  `FileSystemStorage.LoadBlob` (closed `osFile` on the crc/seek error paths; success path returns it
  as the reader, so no blanket defer) + added 2 unit tests for the previously-untested `LoadBlob`.
  Suite stays green; vet clean. PR #13 for #12, branch auto-improve/12-loadblob-fd-leak. (3rd
  security/robustness increment in window=5; HWR feature work #7–#11 landed in between, so not a
  pure run of this axis — no diminishing returns observed, metrics still moving on real bugs.)

## Failed / rejected approaches (do not blindly retry)
_Record what was tried and why it failed so the loop doesn't loop._
- (none yet)
- NOTE (not a failure, a gate): `pnpm -C ui audit` red is NOT loop-actionable — fixing it means
  upgrading deps (vite/ws/esbuild/...), which is human-gated. Don't re-open this as a loop ticket;
  if surfaced, flag for a human only.

## Decisions & notes
_Durable choices worth remembering (e.g. "library X chosen over Y because …")._
- 2026-06-24 — Convention: outbound `http.Client`s carry a 30s timeout (originated in
  `internal/integrations/ics.go:98`). New outbound calls should match this.
- 2026-06-24 — Default branch is `master` (not `main`); treat `master` as the protected branch.

## Iteration log
_One line per run. Newest at top._
- 2026-06-26 — phase: auto — ticket #12 — outcome: PR #13 opened (close blob file on LoadBlob
  error paths → fix FD leak; +2 LoadBlob unit tests) → master. Empty backlog at start; filed #12
  during scouting then implemented it. security/robustness+correctness axis. branch
  auto-improve/12-loadblob-fd-leak. 2 files / +4 lines code + new test file, all Go gates green.
  Noted baseline drift (PRs #6–#11 merged; UI-lint now green) and that the ui-audit gate is red
  but human-gated (deps).
- 2026-06-25 — phase: auto — ticket #5 — outcome: PR #6 opened (fix 2 baseline eslint errors → UI-lint
  gate green; React.FC<any>→React.ComponentType, drop unused `e`) → master. Backlog had only #5;
  code-cleanliness axis. branch auto-improve/5-ui-lint. 2 files / 4 lines, all gates green.
- 2026-06-25 — phase: auto — ticket #3 — outcome: PR #4 opened (bound 4 unbounded request-body reads
  with http.MaxBytesReader + 2 tests) → master. Empty backlog at start; also groomed #5 (red UI lint
  baseline). PR: auto-improve/3-bound-request-bodies.
- 2026-06-24 — phase: auto — ticket #1 — outcome: PR opened (HTTP timeouts + HWR body close + webhook
  test) — PR: auto-improve/1-http-timeouts → master (merged as #2)
