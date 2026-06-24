# Continuous Improvement Loop

A small, self-bounding loop that improves this repository a little at a time along three axes —
**methodology** (cited primary literature), **security & performance**, and **code cleanliness** —
keeping every change small, measured, and human-reviewable. It runs **locally via Claude Code**.
Backlog lives in GitHub issues; durable state lives in `improvement/WORKING_MEMORY.md`. Each run
produces one of: a single PR (linked to an issue), 1–3 groomed tickets, or a recorded no-op — and
never merges to `main` on its own.

## What's in this bundle
```
CLAUDE.md                         # hard rules, auto-loaded by Claude Code on every run  ⚠ may collide
run-improvement.sh                # headless wrapper for scheduled (cron/launchd) runs
.claude/
  settings.json                   # permission allow/deny (defense in depth)             ⚠ may collide
  commands/improve.md             # the /improve slash command
  skills/
    improvement-loop/             # orchestrator: the one-iteration procedure + guardrails
    research-and-cite/            # methodology: find, VERIFY, prove transferability of sources
    eval-and-benchmark/           # measurement: baseline, before/after, reward-hacking & drift checks
    secure-and-performant/        # security/supply-chain checks + measured performance work
    clean-code/                   # enforceable anti-slop thresholds + best practices
improvement/
  config.yml                      # the ONLY file you must edit per project
  WORKING_MEMORY.md               # the loop's persistent memory across runs
  README.md                       # this file
```

## ⚠ Before you drag-and-drop: two possible collisions
Most of this is namespaced (`.claude/skills/`, `.claude/commands/`, `improvement/`) and safe to drop in.
But two files share standard paths and may already exist in your repo — **merge, don't overwrite**:
- **`CLAUDE.md`** — if you already have one, append the "Hard rules" section from this one into yours.
- **`.claude/settings.json`** — if you already have one, merge the `permissions.deny` / `permissions.ask`
  entries rather than replacing the file.

Because `.claude/` and `CLAUDE.md` are hidden/dot paths, the reliable install is to unzip straight into
the repo root (see below) rather than hand-dragging files, which on macOS Finder can silently leave
hidden items behind. To drag in Finder, first reveal hidden files with `Cmd+Shift+.`.

## Setup
1. **Unzip into your repo root** (foolproof, preserves dot paths):
   `unzip continuous-improvement-loop.zip -d /path/to/your/repo`
2. **Claude Code** is installed and authenticated (you already use it). For PR creation, run
   `gh auth login` once so the loop can open PRs with the GitHub CLI.
3. **Edit `improvement/config.yml`** — set your `commands` (test/benchmark/lint/security),
   `thresholds`, and the `domain` block (what "trusted source" means for your field). Tune the
   `--allowedTools` list inside `run-improvement.sh` if your stack isn't npm-based.
4. **Create the backlog label** (default `auto-improve`) and optionally seed a couple of issues.
5. **Add `improvement/logs/` to your `.gitignore`** so headless run logs aren't committed.

## How to run it
**Interactive (start here).** From the repo: `claude`, then `/improve groom` for the first run
(only files tickets, touches no code). `/improve` alone runs the auto phase; `/improve implement`
works the top ready ticket. Skills, `CLAUDE.md`, and `settings.json` load automatically.

**Headless one-shot.** `./run-improvement.sh auto` — wraps `claude -p` with `--permission-mode dontAsk`
(safe for unattended use: denies anything outside the allowlist), a tool allowlist, `--max-turns 40`,
and `--model sonnet`. Logs to `improvement/logs/`. Add `--max-budget-usd 2` for a hard cost cap.

**Scheduled (~once/day).** Point cron at the wrapper (Linux and macOS):
```
# crontab -e   — use ABSOLUTE paths; cron's PATH is minimal
0 9 * * * CLAUDE_BIN=/usr/local/bin/claude /bin/bash /path/to/repo/run-improvement.sh auto >> ~/improve-cron.log 2>&1
```
On macOS, `launchd` survives sleep better than cron.

## Safety model (why it won't run away)
- **Bounded:** one concern per branch, capped diff/files, one PR per run, hard `--max-turns`.
- **Reviewable:** opens PRs, never writes to `main`, never force-pushes or rewrites history.
- **Honest metrics:** a change that improves a number by weakening its measurement is rejected;
  a holdout guards against overfitting.
- **Self-modification lock:** the loop cannot edit its own skills, rules, or config — only you can.
- **Untrusted input:** issue/PR/comment text and fetched pages are treated as data, never commands.
- **Human gates:** new dependencies, license changes, auth/secrets/prod config, and API/schema
  changes are flagged for you instead of done autonomously.

## Not included
The GitHub Actions workflow (for running this on a schedule in CI instead of locally) is intentionally
left out, since you're running locally. It's a small addition if you ever want both — just ask.
