# Local fork of rmc-go

This is a vendored copy of [`github.com/joagonca/rmc-go`](https://github.com/joagonca/rmc-go)
**v1.1.1** (MIT licensed), wired in via a `replace` directive in the repo-root `go.mod`.
It carries one local patch that upstream lacks.

## Patch: word-wrap for typed text

Upstream renders each typed-text paragraph (`RootText` / `Text` blocks, including the
text the reMarkable produces from "Convert to text") as a single unwrapped line that
starts at the block's `PosX`. The block's `Width` is only used to size the page, never
to wrap. A long paragraph therefore runs off the right edge of the page in the PDF
export and the web preview, even though the tablet itself wraps it.

`export/pdf_cairo.go` was changed to wrap each paragraph to the text block's width:

- `wrapParagraph`, `maxCharsPerLine`, `fontPointSize`, `textWrapWidth`,
  `paragraphLines` — new helpers that break a paragraph into visual lines that fit the
  block width (character-budget based, so page sizing and drawing share one
  calculation without a measuring pass).
- `drawTextCairo` — draws one line per wrapped line, advancing the baseline per line.
- `calculatePageDimensions` — counts wrapped lines (not paragraphs) so the page stays
  tall enough, and sizes width from `textWrapWidth` rather than a possibly-zero `Width`.

The SVG exporter (`export/svg.go`) is intentionally **not** patched: the app preview and
PDF export both flow through the Cairo path; SVG wrapping can follow if needed.

Tests proving the behavior live in the main module:
`internal/storage/exporter/textwrap_cairo_test.go`.

## Re-syncing with upstream

1. Re-copy the upstream module over this directory at the new version.
2. Re-apply the wrap patch (see `git log`/`git diff` for this directory, or search for
   `wrapParagraph` in `export/pdf_cairo.go`).
3. Bump the version in the repo-root `go.mod` `require` line; keep the `replace`.
