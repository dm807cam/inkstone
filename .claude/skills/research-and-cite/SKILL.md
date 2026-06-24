---
name: research-and-cite
description: >-
  Improve a project's method using trusted primary literature, then prove the change is real:
  verify each source exists and says what is claimed, confirm the method transfers to THIS
  project's data and constraints, and cite it in code and the PR. Use for any methodological
  upgrade, algorithm change, or "is there a better approach in the literature" task.
---

# Research and Cite (methodology axis)

A method being published is **not** evidence it helps this project. This skill turns "I read a
paper" into a defensible, cited, *transferable* change. Hallucinated or misapplied citations are
the main failure mode here — guard against them explicitly.

## 1. Find trusted, primary sources
Prefer, in order: peer-reviewed papers and standards bodies; official documentation and
specifications of the libraries/tools in use; reputable reference implementations. The domain
profile in `improvement/config.yml` lists the sources to trust for *this* field (e.g. specific
journals/venues, RFCs, vendor docs). Avoid blogs, forums, and SEO content except as pointers to
a primary source you then read directly. Prefer the most recent authoritative version.

## 2. Verify every source (anti-hallucination gate)
For each source you intend to cite, confirm **all** of:
- **It exists.** Resolve a stable identifier — DOI, arXiv ID, RFC number, or a canonical URL —
  by fetching it. If you cannot resolve it, you may not cite it. Do not invent authors, titles,
  years, or identifiers.
- **It says what you claim.** Locate the specific result/section that supports your change.
  Paraphrase it in your own words; quote at most a short phrase if exact wording matters.
- **It is current/uncontested enough.** Note retractions, errata, or a clear newer consensus.

## 3. Confirm transferability to *this* project (the step people skip)
Before changing code, state plainly whether the method's preconditions hold here:
- Do our **data** (size, distribution, dimensionality, noise, licensing) and **constraints**
  (latency, memory, online/offline, language/runtime) match the paper's setting?
- What does the source assume that we cannot guarantee? What breaks if those assumptions fail?
- Is there a cheaper baseline that captures most of the benefit?
If the method does not transfer, **do not implement it** — record the finding (so it isn't
re-researched) and stop, or propose an adapted variant with its own justification.

## 4. Implement with the method made explicit
Hand the actual change to `clean-code` and `secure-and-performant`. In the code, add a focused
comment naming the method and the citation so the *why* lives next to the *what*. Keep the change
within the loop's diff budget; a method upgrade is still one bounded increment.

## 5. Cite in code and in the PR
- **In code:** a brief comment at the relevant function — method name + short citation
  (e.g. `# Welford's online variance — Welford 1962, doi:10.1080/00401706.1962.10490022`).
- **In the PR body:** a short "Method & evidence" section: what changed, the source (with stable
  identifier), the transferability argument, and the measured effect (from `eval-and-benchmark`).
- Maintain a running `CITATIONS.md` (path set in `config.yml`) so the project's methodological
  provenance is auditable in one place. One entry per source: identifier, claim used, where applied.

## Output contract
A citation is acceptable only if it is **resolvable**, **supports the specific claim**, and the
method is **shown to transfer**. Anything failing these is dropped, not "best-guessed".
A methodological PR must always pass `eval-and-benchmark` — a better method that loses on the
eval suite is not better here.
