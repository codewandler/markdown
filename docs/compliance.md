# Spec Compliance

## Summary

| Spec | Pass | Total | Pct |
| --- | ---: | ---: | ---: |
| CommonMark 0.31.2 | 652 | 652 | **100.0%** |
| GFM 0.29 spec.txt | 663 | 672 | 98.7% |
| GFM extensions.txt | 16 | 30 | 53.3% |
| GFM regression.txt | 15 | 26 | 57.7% |
| **GFM total** | **707** | **728** | **97.1%** |

## Test corpora

We test against three files from the
[cmark-gfm](https://github.com/github/cmark-gfm) repository:

| File | Examples | Description |
| --- | ---: | --- |
| `test/spec.txt` | 672 | The official GFM specification (CommonMark 0.29 base + 24 GFM extension examples) |
| `test/extensions.txt` | 30 | Additional extension tests written by the cmark-gfm maintainers |
| `test/regression.txt` | 26 | Regression tests for specific bugs fixed in cmark-gfm |

Per-example extension dispatch: the spec.txt examples declare which
extensions they require (e.g. `autolink`, `tagfilter`). We enable only
the declared extensions per example, matching the official `spec_tests.py`
behavior. The extensions.txt and regression.txt files assume all
extensions are active.

## Remaining GFM failures (21)

### Emphasis Rule 13 — spec divergence (9 examples)

CommonMark 0.31.2 and GFM 0.29 disagree on Rule 13 ("minimize nesting
depth"). For example, `****foo****`:

- CommonMark 0.31.2: `<strong><strong>foo</strong></strong>` (nested)
- GFM 0.29: `<strong>foo</strong>` (collapsed)

We follow CommonMark 0.31.2 (the newer spec). These 9 examples are
**unfixable without breaking CommonMark compliance**.

Affected: spec.txt examples 398, 426, 434, 435, 436, 473, 474, 475, 477.

### Footnotes — not implemented (7 examples)

GFM footnotes (`[^ref]` / `[^ref]: definition`) are not yet supported.
The parser treats `[^...]` as literal text.

Affected: ext#23, ext#24, ext#25, reg#13, reg#20, reg#21, reg#22.

### Task list format — spec conflict (3 examples)

The GFM spec.txt and extensions.txt disagree on task list checkbox
attribute order:

- spec.txt: `<input disabled="" type="checkbox">`
- extensions.txt: `<input type="checkbox" disabled="" />`

We match spec.txt. These 3 examples are **unfixable without breaking
spec.txt compliance**.

Affected: ext#28, ext#29, ext#30.

### Table reference links (1 example)

Reference links in table cells are not resolved when the link reference
definition appears after the table. Table cell inline parsing is not
deferred like paragraph parsing.

Affected: ext#13.

### Autolink edge cases (1 example)

The extensions.txt autolink mega-test (ext#19) covers many edge cases
including `mailto:`, `xmpp:` scheme recognition, email followed by `/`,
and www domain underscore validation in the last two segments. Several
sub-cases within this single example still fail.

## Comparison with goldmark

| Corpus | ours | goldmark |
| --- | ---: | ---: |
| spec.txt | **663/672** | 655/672 |
| extensions.txt | 16/30 | **21/30** |
| regression.txt | **15/26** | 14/26 |
| **Total** | **707/728** | 690/728 |

We lead on spec.txt (+8) and regression.txt (+1). Goldmark leads on
extensions.txt (+5), primarily due to footnote support (3 examples)
and task list format matching extensions.txt (2 examples).

## Reproduction

```bash
# Run all three GFM corpora against all competitors
cd competition && go test -run TestGFMFullSuite -v ./benchmarks/

# Run CommonMark compliance
go test -run TestHTMLCommonMarkCompliance -v .

# Run the full competition pipeline
task competition:compliance
```
