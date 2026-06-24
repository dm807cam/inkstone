---
name: clean-code
description: >-
  Reduce slop and enforce best practices: modular, readable, well-named code with useful comments,
  no dead code, and no duplication. Applies enforceable thresholds (complexity, duplication,
  dead-code, file/function size) that block a change, plus the rule to functionalise anything used
  more than twice. Use while writing any change and for code-cleanliness audits.
---

# Clean Code (cleanliness / anti-slop axis)

"Reduce slop" only means something with teeth. This skill applies thresholds that **block** a
change rather than vibes. Thresholds live in `improvement/config.yml` so they're tunable per project.

## Enforceable gates (a change must not breach these)
- **Cyclomatic complexity** per function ≤ `max_complexity`. Above it, decompose.
- **Duplication**: no new copy-pasted block beyond `max_duplication`. The rule of thumb is strict —
  **if logic is used more than twice, functionalise it** into a single named unit and call it.
- **Dead code**: no unreachable code, unused exports, commented-out blocks, or orphaned files left
  behind. Removing dead code is itself a valid increment (behind a green suite).
- **Size**: function and file length within the limits in config; long units are a smell to split.
- **Lint/format/type-check**: zero new violations; the change must pass the project's configured
  linters, formatter, and type checker.

## Best practices to apply while writing
- **Modularity & single responsibility.** Small units with one job; clear module boundaries; depend
  on interfaces, not internals. Pure functions where practical; isolate side effects.
- **Naming.** Intention-revealing names; no abbreviations that aren't domain-standard.
- **Comments that earn their place.** Explain *why*, not *what*; document non-obvious decisions,
  invariants, units, and any cited method (hand method citations to `research-and-cite`). Delete
  comments that merely restate the code.
- **Consistency.** Match the existing patterns and style of the codebase; don't introduce a second
  way to do something that already has one.
- **Tests track behavior.** New behavior gets a test; refactors keep tests green without weakening
  them (weakening tests to pass is a reward-hack — see `eval-and-benchmark`).

## Refactor discipline
A cleanup increment must be **behavior-preserving** and proven so by `eval-and-benchmark`
(metrics held flat, suite green). Keep refactors separate from feature/method changes — one
concern per branch — so review and rollback stay clean. Stay within the loop's diff budget;
a sprawling "tidy everything" diff is itself slop.

## Output contract
A change is clean-gate-ready only when every enforceable gate above passes and the diff is the
minimum needed for the stated concern. For an audit: file ranked, checkable tickets for the worst
offenders (the most duplicated logic, the most complex function, the largest dead-code island).
