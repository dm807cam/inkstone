# Working Memory

The improvement loop's persistent state across runs. The loop reads this at the start of every
run and updates it at the end. Keep it concise: append iteration entries, keep ONE current
baseline block, and prune notes older than ~30 days unless still relevant. This file is committed
to the repo via each PR (or directly when a run only grooms/records).

> Tip: keep this file small. If it grows large, move resolved history to `improvement/ARCHIVE.md`.

## Current baseline
_Last measured: 2026-07-02 on commit f3aca5f (origin/master). Drift since f32faf8: PRs #25 (export visible-name filename, #22) & #24 (grooming record) merged. All gate metrics unchanged (go test ./... PASS, go vet clean; ui-audit still RED = human-gated deps). Note: `gofmt -l internal cmd` lists 7 pre-existing files with formatting drift (app.go, staticfs.go, config.go, localfs.go, broker.go, version_test.go, models.go) — NOT loop-introduced; leave for a human, do not reformat unrelated files._

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
- Month: 2026-07
- Increments merged (2026-06): 7 (PRs #2, #4, #6, #13, #15 → master; #18 root-blob FD-leak, #19 OCR export)
- PRs open: #21 (OCR export error status, #20) · #27 (this run — uploadDoc missing-meta panic→400, #26, draft)
- Merged since last run: #24 (grooming record) & #25 (export visible-name filename) → master @f3aca5f
- Month: 2026-06
- Increments merged: 6 (PR #2 → master @038fee7; PR #4 → master @3e05326; PR #6 → master @fae3885;
  PR #13 → master @ac5456c — LoadBlob FD-leak fix; PR #15 → master — screenshare payload panic;
  PR #18 → master @604f534 — root-blob FD-leak fix for #17)
- PRs open: 1 (#21 — OCR export error-status fix for #20, draft, awaiting human review)
- This run (2026-06-30): grooming only — filed tickets #22 (export filename) and #23 (metadata
  stub); 0 fix PRs (max_iterations_per_run=1 respected; the only prior backlog item #20 is already
  covered by the open draft PR #21).
- Approx. tokens used: n/a (monthly_token_budget = 0, no cap)

## Metric trend (for diminishing-returns detection)
_Most recent increments and their effect on the targeted metric._
- 2026-07-02 — axis: correctness/robustness (guard parity) — Δ: `uploadDoc` (`internal/app/handlers.go`)
  indexed `form.Value["meta"][0]` with no length guard, so a device multipart upload missing the
  `meta` field panicked (→ HTTP 500 via gin Recovery + stack-trace log) instead of the intended 400.
  The lone unguarded slice-index in the function: the same handler already guards `form.File["file"]`
  (len<1→400), `uploadDocV2` guards its meta header, and the integration-message handler guards
  `attachment`/`data`. Added the missing `len(form.Value["meta"]) < 1` guard (kept the present-but-empty
  check) + 1 new test that POSTs a meta-less multipart form and asserts 400. Anti-reward-hacking:
  the test panics (`index out of range [0] with length 0`) on the pre-fix handler and passes after
  (verified via git-stash of the handler). 2 files / +61/-1. issue #26; branch
  auto-improve/26-uploaddoc-meta-panic. PR #27 (draft). Note: robustness-adjacent but a distinct,
  real, previously-unmined defect (a specific unguarded index), and it alternates with #22's
  correctness/clean-code — not a repeat of the io.ReadAll-bounding/nil-map class. window=5 OK.
- 2026-07-01 — axis: correctness/UX (clean-code) — Δ: `getDocument` (`internal/ui/handlers.go`) named
  `.rmdoc/.txt/.md` downloads by the opaque document UUID; non-browser clients (curl/rmapi) that honor
  `Content-Disposition` got files like `a1b2c3d4-….txt` instead of the visible notebook name. Resolve
  docid→name via `GetDocumentTree(uid)` (no `backend` interface change — the lightest option), fall back
  to docid when unresolved, and emit an ASCII `filename=` + RFC 5987 `filename*=UTF-8''` (RFC 6266) so
  non-ASCII titles survive. +2 handler tests (visible-name + fallback); the visible-name test fails
  pre-fix (asserted the old docid header) and passes after — anti-reward-hacking verified. 2 files /
  +150/-4 (at the 150-line cap). issue #22; branch auto-improve/22-export-visible-filename. PR #25 (draft).
  Deliberately a correctness/clean-code increment, not security/robustness (over-mined; window=5).
- 2026-06-30 — axis: grooming (no metric movement) — Δ: scouting pass over the newest code (the #19
  LLM OCR-export feature: `internal/storage/exporter/ocr.go`, `internal/hwr/{llm,render}.go`,
  `internal/storage/fs/blobstore.go` `ExportOCR`, `internal/ui/handlers.go` `getDocument`). Verified
  the one real OCR bug is already in flight (PR #21 for #20) and that OCR export correctly honors
  cPages page order (`V6PageData` is keyed by index into `Content.OrderedPages()` in
  `ArchiveFromHashDoc`, and `ExportOCR` maps `pages[idx]` the same way — no ordering regression vs
  the #16 cPages fix). Filed two lower-priority tickets: #22 (export `Content-Disposition` uses the
  opaque docid, not the visible name — only affects non-browser API clients; the web UI overrides
  the filename client-side) and #23 (the routed `/documents/:docid/metadata` endpoint is a stub
  returning `200 "TODO"`). No fix PR this run — no high-confidence, in-budget, non-human-gated
  defect remained un-addressed. (security/robustness axis intentionally not pursued: over-mined per
  window=5; the new findings are correctness/clean-code.)
- 2026-06-28 — axis: code cleanliness (dedup) + correctness — Δ: fixed an FD leak on the device-sync
  hot path — `syncGetRootV3`/`syncGetRootV4` (`internal/app/handlers.go`) read the root blob via
  `LoadBlob` (an `io.ReadCloser` the caller owns) with `io.ReadAll` but never `Close()`d it, while
  the identical `blobStorageRead`/`GetRootIndex` both defer Close. Extracted a `loadRootHash` helper
  that loads+reads+ALWAYS closes, returning `fs.ErrorNotFound` verbatim so V3 (404) / V4 (200) keep
  their new-account behaviour; removes the copy-paste that caused the leak. +2 tests (close-count guard
  that fails pre-fix + not-found behaviour). 2 files / +124/-19. issue #17; branch
  auto-improve/17-root-blob-fd-leak. PR #18 (draft). Deliberately
  switched off the pure security/robustness axis per the window=5 note; this increment is dedup-led.
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
- 2026-06-27 — axis: security/robustness + correctness — Δ: fixed a nil-map-write panic in
  `screenshareSendAnswer` (`internal/ui/handlers.go`): the client `payload` was unmarshalled into a
  nil map with the error ignored, then written to — a missing/null/non-object payload panicked on
  untrusted input. Now validated up front (400 before forwarding); happy path still 202. +1 new test
  file (3 reject cases + 1 accept). PR #15 for #14, branch auto-improve/14-screenshare-payload-panic.
  (4th security/robustness loop increment overall; feature PRs #7–#11 interleaved, so not a pure run
  of this axis — still finding distinct real bugs, no diminishing returns. If the NEXT increment also
  lands here, deliberately switch axes per cadence.diminishing_returns_window=5.)
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
- 2026-06-27 — Testing gin handlers directly (gin.CreateTestContext, no engine/Recovery): `c.Status(code)`
  is buffered and NOT flushed to the httptest recorder unless a body is written, so assert on
  `c.Writer.Status()` (gin's tracked status), not `recorder.Code`. `internal/ui` handlers can be unit
  tested by constructing a minimal `&ReactAppWrapper{...}` with only the fields the code path touches
  (e.g. `roomManager` via `screenshare.NewRoomManager()`, `h` via `hub.NewHub()`); leave `mqtt` nil to
  skip the MQTT branch. A panic in a directly-called handler propagates to the test (no Recovery), which
  makes nil-map/nil-deref regressions easy to pin.

## Iteration log
_One line per run. Newest at top._
- 2026-07-02 — phase: auto — ticket #26 (filed this run) — outcome: PR #27 opened (draft) → master.
  Sensed baseline on f3aca5f: `go test ./...` PASS, `go vet ./...` clean (built ui/dist first for the
  embed) — unchanged vs recorded baseline; drift = #24 + #25 merged. Backlog at start: #20 (in flight
  as PR #21), #23 (human-gated remove-vs-implement) — no non-gated, un-in-flight ticket, so scouted.
  Found + filed #26: `uploadDoc` indexed `form.Value["meta"][0]` unguarded → panic/500 on a meta-less
  multipart upload (the only unguarded slice-index in the function; siblings all guard). Implemented
  the missing length guard + 1 test (panics pre-fix, 400 after). correctness/robustness axis. branch
  auto-improve/26-uploaddoc-meta-panic. 2 files / +61/-1; go test + go vet + gofmt(my files) green;
  ui-audit still red (human-gated deps, untouched). #23 left for a human; #20 covered by PR #21.
- 2026-07-01 — phase: auto — ticket #22 — outcome: PR #25 opened (draft) → master. Backlog at start:
  #20 (in flight as draft PR #21), #22, #23 (#23 defers a remove-vs-implement choice to a human).
  Selected #22 (concrete, non-gated). Named exports by visible name w/ RFC 5987 filename* + docid
  fallback; +2 handler tests. correctness/clean-code axis. branch auto-improve/22-export-visible-filename.
  2 files / +150/-4; go test + go vet + gofmt green; ui-audit still red (human-gated deps, untouched).
- 2026-06-30 — phase: auto — outcome: GROOMING (filed #22 + #23), no fix PR. At run start the only
  open auto-improve ticket (#20) was already covered by open draft PR #21, so there was no
  un-addressed backlog item to implement. Sensed baseline on f32faf8: `go test ./...` PASS (all pkgs),
  `go vet ./...` clean — unchanged vs recorded baseline (drift = #18 + #19 merged). Scouted the new
  #19 OCR-export feature: confirmed page ordering honors cPages and the render/transcribe path is
  sound; the one real bug is already in PR #21. Filed #22 (export filename = opaque docid; low-pri,
  UI-overridden) and #23 (dead `/metadata` stub → `200 "TODO"`). WORKING_MEMORY update committed on
  branch auto-improve/22-groom-export-filename → draft PR (no direct master push per CLAUDE.md #1).
- 2026-06-28 — phase: auto — ticket #17 (filed this run) — outcome: PR #18 opened (draft) → master.
  Empty backlog at start; scouted, filed #17, implemented it. Closed an FD leak in syncGetRootV3/V4
  by extracting a `loadRootHash` helper that always closes the LoadBlob reader (dedup removes the
  copy-paste that dropped the close). 2 files / +124/-19; go test + go vet + gofmt all green; new
  close-count test fails pre-fix, passes after (anti-reward-hacking verified). NOTE: GitHub MCP token
  expired mid-run; recovered after a brief retry, so #17/#18 went through. Baseline → 77d4f40 (#15/#16).
- 2026-06-27 — phase: auto — ticket #14 — outcome: PR #15 opened (reject missing/null/non-object
  screenshare `payload` with 400 instead of panicking on a nil-map write; +1 ui test file) → master.
  Empty backlog at start; scouted, filed #14, then implemented it. security/robustness+correctness
  axis. branch auto-improve/14-screenshare-payload-panic. 2 files / +69/-2, all Go+UI gates green;
  ui-audit still red (human-gated deps, untouched). Baseline refreshed to ac5456c (PR #13 merged).
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
