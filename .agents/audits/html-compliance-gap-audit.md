# HTML Compliance Gap Audit

**Date:** 2026-04-29
**Baseline:** 469/652 (71.9%) HTML compliance, 627/652 (96.2%) event-level
**Delta:** 183 examples pass event-level but fail HTML output

## Summary by Root Cause

| Root Cause | Count | Fixable by | Effort |
|---|---:|---|---|
| **other (HTML block rendering)** | 63 | parser | high — HTML block types 1-7 |
| **parser:emphasis-algorithm** | 46 | parser | medium — emphasis across soft breaks, nesting |
| **parser:emphasis-placement** | 21 | parser | medium — overlaps with HTML block recognition |
| **parser:link-not-recognized** | 10 | parser | medium — ref defs, empty dest, forward refs |
| **parser:inline-raw-html** | 10 | parser | medium — inline HTML tag passthrough |
| **parser:code-block-content** | 8 | parser | low — tab expansion in code blocks |
| **parser:nested-blockquote** | 6 | parser | high — blockquote stack rearchitecture |
| **parser:gfm-autolink-in-commonmark** | 5 | parser | low — disable GFM autolinks for CM tests |
| **parser:html-block-not-recognized** | 5 | parser | medium — HTML comment/tag recognition |
| **renderer:whitespace** | 3 | renderer | low — trailing space in code spans |
| **parser:list-structure** | 3 | parser | medium — list interruption rules |
| **parser:ref-link** | 2 | parser | medium — image in ref link, autolink in ref |
| **parser:image-not-recognized** | 1 | parser | low — inline raw HTML precedence |

## Actionable Categories

### 1. renderer:whitespace (3 examples) — FIX NOW

The only renderer bugs left. All in code spans:

- **331**: `\`  \`\`  \`` — trailing space stripped inside code span
- **334**: `\`  \`` — space-only code span stripped to empty
- **336**: `` `\nfoo \n` `` — trailing space stripped inside code span

**Root cause:** Our trailing-space stripping fires on text before
paragraph close, but code span content should preserve spaces. The
stripping is too aggressive — it strips inside `<code>` text events
that happen to precede a paragraph close.

**Fix:** Skip trailing-space stripping when `ev.Style.Code` is true.

### 2. parser:gfm-autolink-in-commonmark (5 examples) — LOW EFFORT

The parser recognizes GFM extended autolinks (bare URLs, emails) even
in CommonMark context. CommonMark examples 608, 611, 612 expect bare
URLs to be plain text. Example 602 expects `<url with space>` to not
be an autolink.

**Options:**
- (A) Add a parser option to disable GFM autolinks. The HTML
  compliance test uses it. Terminal rendering keeps GFM on.
- (B) Accept as known gap — GFM autolinks are a feature, not a bug.
  The event-level tests already account for this.

**Recommendation:** (B) for now. These are GFM extensions working as
designed. The compliance test can exclude them. Revisit if we add a
strict CommonMark mode.

### 3. parser:emphasis-algorithm (46 examples) — MEDIUM EFFORT

Two sub-categories:

**a) Emphasis across soft breaks (4 examples: 81, 82, 638, 639):**
The parser splits emphasis at line boundaries. `*foo\nbar*` produces
two separate `<em>` spans instead of one spanning the soft break.
This is a parser inline-parsing issue — emphasis delimiters don't
match across soft break tokens.

**b) Emphasis nesting/rule-of-three (42 examples):**
The CommonMark emphasis algorithm edge cases. Already documented in
the original gap audit. The parser's `resolveEmphasis` needs the
full CommonMark "process emphasis" algorithm.

### 4. parser:emphasis-placement (21 examples) — OVERLAPS WITH HTML BLOCKS

Most of these (8/21) are HTML block examples where emphasis appears
inside or after an HTML block. The parser doesn't recognize the HTML
block, so the emphasis context is wrong. Fixing HTML block recognition
would resolve these automatically.

### 5. other (63 examples) — HTML BLOCK RECOGNITION

The largest category. 31 are in the HTML blocks section, the rest are
scattered across sections where HTML blocks appear inline. The parser
recognizes some HTML blocks (types 6-7) but misses:

- Type 1: `<pre`, `<script`, `<style`, `<textarea` (start tags)
- Type 2: `<!--` comments (partially recognized)
- Type 3: `<?` processing instructions
- Type 4: `<!` declarations
- Type 5: `<![CDATA[`
- Leading whitespace before HTML block start tags

Many of these produce `<br />` artifacts because the parser treats
HTML tag lines as paragraph content with hard line breaks.

### 6. parser:inline-raw-html (10 examples) — MEDIUM EFFORT

The parser doesn't pass through inline HTML tags like `<a href="...">`,
`<span>`, `<!-- comment -->` in paragraph text. Instead it escapes
them. The AGENTS.md says inline raw HTML parsing is in scope.

### 7. parser:code-block-content (8 examples) — TAB EXPANSION

Tab characters in code blocks should expand to spaces (to the next
tab stop at column multiple of 4). The parser strips tabs or doesn't
expand them correctly. Affects examples 5, 6, 7, 9 (Tabs section)
and fenced code examples 124, 128, 143.

### 8. parser:nested-blockquote (6 examples) — HIGH EFFORT

`> > > foo` should produce triple-nested blockquotes. The parser only
handles single-level `>`. Already documented in roadmap as requiring
blockquote stack rearchitecture.

### 9. parser:link-not-recognized (10 examples)

- Multiline ref def titles (#196)
- Empty link destination `<>` (#200)
- Forward ref in heading (#214)
- Ref def in blockquote (#218)
- Various ref link edge cases

### 10. parser:list-structure (3 examples)

- HTML block inside list item (#175)
- Marker-only lines not treated as empty items (#285, #367)

## Priority Order for Maximum HTML Compliance Gain

| Priority | Category | Examples | Effort | New total |
|---|---|---:|---|---:|
| 1 | renderer:whitespace | 3 | 15 min | 472 |
| 2 | parser:emphasis across breaks | 4 | 1-2 hr | 476 |
| 3 | parser:tab expansion | 8 | 2-3 hr | 484 |
| 4 | parser:inline-raw-html | 10 | 3-4 hr | 494 |
| 5 | parser:HTML block recognition | ~68 | 4-6 hr | ~562 |
| 6 | parser:emphasis algorithm | 42 | 4-6 hr | ~604 |
| 7 | parser:link edge cases | 12 | 2-3 hr | ~616 |
| 8 | parser:nested-blockquote | 6 | 4-6 hr | ~622 |
| 9 | parser:list-structure | 3 | 1-2 hr | ~625 |
| 10 | parser:gfm-autolink | 5 | accept gap | 625 |

**Estimated ceiling:** ~625/652 (95.9%) with all fixes.
**Remaining ~27:** GFM autolink conflicts (5), forward references (5-10),
deeply nested container edge cases, and misc parser limitations.

## Quick Win: renderer:whitespace (3 examples)

Fix trailing-space stripping to skip code span content:

```go
// In render loop, before stripping:
if !r.inCode && !r.inHTML && !ev.Style.Code && i+1 < len(events) && ...
```

This is the only remaining renderer bug. Everything else is parser.
