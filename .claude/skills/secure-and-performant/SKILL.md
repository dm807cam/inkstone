---
name: secure-and-performant
description: >-
  Keep the project production-safe and reasonably fast. Run security and supply-chain checks
  (static analysis, dependency vulnerabilities, secret scanning, license audit) and find/justify
  performance improvements with measured before/after numbers. Use when implementing any change
  and for periodic security/performance audits of the codebase.
---

# Secure and Performant (security & performance axis)

This skill makes a change *production-safe* and *reasonably performant*. Apply it while writing
the change, not as an afterthought. All commands and thresholds come from `improvement/config.yml`.

## Security & supply-chain (every change)
Run, and treat new findings introduced by the change as blocking:
- **Static analysis / SAST** for the project's language(s); fix or do not introduce findings.
- **Dependency vulnerabilities**: scan the lockfile; do not add a dependency with known critical/high
  CVEs. **Adding or upgrading any dependency is a human-gated action** — open a flagged ticket/PR,
  never silently pull in new supply chain.
- **Secret scanning**: never commit credentials, tokens, or keys; never print secrets to logs.
- **License audit**: a new dependency's license must be compatible with the project's
  (`allowed_licenses` in config). Incompatible or unknown license ⇒ stop and flag.
- **Input handling**: validate and bound external input; avoid injection sinks (shell, SQL, path,
  deserialization); fail closed. Apply least privilege to anything the code can reach.

## Production-safety checklist
Before a change is gate-ready: errors are handled and surfaced, not swallowed; no unbounded
resource use (memory, connections, recursion); concurrency is safe where relevant; logging is
adequate but leaks nothing sensitive; configuration/feature flags have safe defaults; the change
is observably reversible (clear rollback path noted in the PR).

## Performance (only what you can measure)
Performance work must be evidence-led via `eval-and-benchmark`:
1. **Measure first.** Identify the real hot path with a profiler or benchmark — never optimize on a
   hunch or micro-optimize cold code.
2. **Change one thing**, then re-measure with the same harness; report before → after with variance.
3. **Guard readability.** Accept a complexity increase only when the measured win is worth it, and
   leave a comment explaining the tradeoff. Prefer algorithmic wins (better complexity class) over
   bit-twiddling. A perf change that regresses a security or correctness gate is rejected.

## Periodic audit mode (when grooming the backlog)
Survey the repo for the highest-value security and performance issues and file concrete tickets:
the dependency with the worst CVE exposure, the unvalidated boundary, the documented hot path
with no benchmark, the obvious N+1 or quadratic loop on a large input. Each ticket gets an
acceptance check so a later increment can verify it objectively.

## Output contract
For a change: a security delta (no new findings; CVEs/licenses for any proposed dependency
surfaced for human approval) and, for perf work, measured before→after numbers from
`eval-and-benchmark`. For an audit: ranked, checkable tickets.
