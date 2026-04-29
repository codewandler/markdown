# Plan: Parser Changes to Reach 100% CommonMark Compliance

**Status:** 600/652 (92.0%) → target 652/652 (100%)
**52 failures remaining** — all parser-level in `stream/parser_impl.go`
**HTML renderer is complete with zero renderer bugs.**

---

## Phase 1: Autolinks — CommonMark vs GFM (5 fixes) ⬜
**Examples: 602, 603, 608, 611, 612**
**Difficulty: Easy | Impact: 5 examples**

The parser currently enables GFM autolink literals unconditionally. CommonMark
spec only recognizes `<scheme:uri>` autolinks, not bare `https://` URLs.

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 602 | `<https://foo.bar/baz bim>` | Not autolink (space) | Autolinked `baz` part | `isURIAutolink` allows space |
| 603 | `<https://example.com/\[\>` | `%5C%5B%5C` | `%5C[%5C` | Backslash not percent-encoded |
| 608 | `< https://foo.bar >` | Not autolink (leading space) | Autolinked | `parseAutolink` doesn't reject leading space |
| 611 | `https://example.com` | Plain text | Autolinked | GFM literal autolinks enabled |
| 612 | `foo@bar.example.com` | Plain text | Autolinked | GFM literal autolinks enabled |

### Changes needed:
1. **`isURIAutolink`**: Reject targets containing spaces (602) and leading spaces (608 — already handled by `parseAutolink` since `<` is at pos 0, but the space after `<` means the `>` search finds `baz bim>` which contains space)
2. **`parseAutolink`**: The `>` search at `strings.IndexByte` finds the first `>`, but the target `https://foo.bar/baz bim` contains a space → `isURIAutolink` should reject it. Actually checking: `isURIAutolink` already checks `c <= ' '` — so 602 should already fail. Let me re-check... The issue is that `< https://foo.bar >` — the target is ` https://foo.bar ` with leading space. `isURIAutolink` checks chars after colon, but the leading space is before the scheme. Need to check: the target starts with space, so `isASCIIAlpha(target[0])` returns false → should already fail. **Re-examine**: 608 is being autolinked by `parseAutolinkLiteral`, not `parseAutolink`. The `<` fails as autolink, then `https://foo.bar` gets picked up as a GFM literal.
3. **GFM autolink literals** (611/612): Add a `GFMAutolinks bool` config option (default true for GFM, false for CommonMark). Gate `parseAutolinkLiteral` on this flag.
4. **Percent-encoding** (603): In `parseAutolink`, percent-encode `\` and `[` in the URL destination.

### Files: `stream/parser_impl.go` (autolink functions), `stream/config.go`

---

## Phase 2: Raw HTML Comment Fix (1 fix) ⬜
**Example: 626**
**Difficulty: Easy | Impact: 1 example**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 626 | `foo <!--> foo -->` | `foo <!--> foo --&gt;` | `foo &lt;!--&gt; foo --&gt;` | Comment rejected |

The spec says `<!--` followed by `>` is a valid (empty) HTML comment `<!-->`
and `<!---` followed by `>` is also valid `<!--->`. Our `parseRawHTMLTag`
explicitly rejects these cases.

### Changes needed:
1. **`parseRawHTMLTag`**: Remove the two early-return checks for `<!--` followed by `>` and `<!--` followed by `->`. Instead, just search for `-->` starting from position 4. The spec says the comment text must not start with `>`, start with `->`, contain `--`, or end with `-` — but `<!--> ` and `<!--->` are the *degenerate* cases where the comment text is empty or `-`, which the spec explicitly allows as valid comments.

Actually re-reading the spec more carefully: CommonMark 0.31.2 §6.6 says an HTML comment is `<!--` + text + `-->` where text does not start with `>` or `->`, does not end with `-`, and does not contain `--`. But `<!-->` and `<!--->` are listed as valid tags in the spec examples. Let me check... Example 626 expects `<!-->` to be passed through as raw HTML. So the spec treats `<!-->` as a valid comment (empty text, the `>` is the closing). This means the comment is `<!--` + `` + `>` — wait, that doesn't match `-->`. 

Actually, the HTML spec (not CommonMark) defines comments differently. CommonMark 0.31.2 says: "An HTML comment consists of `<!--` + text + `-->`, where text does not start with `>` or `->`, does not end with `-`, and does not contain `--`." But example 626 shows `<!--> foo -->` producing `<!--> foo --&gt;` — meaning `<!-->` is the complete comment tag, and ` foo -->` is literal text. So `<!-->` is parsed as `<!--` + `` (empty) + `>` — but that's not `-->`. 

Hmm, this suggests the spec changed. Let me look at the actual spec version...

The key insight: `<!-->` is a valid HTML comment per the HTML spec (it's an empty comment). CommonMark 0.31.2 must have updated to match. The rule is: `<!-->` and `<!--->` are valid comments as special cases.

### Changes needed:
1. In `parseRawHTMLTag`, change the `<!--` handling: if `text[4] == '>'`, return `text[:5]` as a valid comment. If `text[4] == '-' && text[5] == '>'`, return `text[:6]` as valid comment. Then search for `-->` as before.

### Files: `stream/parser_impl.go` (`parseRawHTMLTag`)

---

## Phase 3: Image Alt Text Stripping (7 fixes) ⬜
**Examples: 573, 574, 575, 576, 577, 585, 589**
**Difficulty: Medium | Impact: 7 examples**

| # | Input | Expected alt | Got alt | Root cause |
|---|-------|-------------|---------|------------|
| 573 | `![foo *bar*]` (ref) | `foo bar` | `foo *bar*` | Markup not stripped |
| 574 | `![foo ![bar](/url)](/url2)` | `foo bar` | `foo ![bar](/url)` | Nested image not stripped |
| 575 | `![foo [bar](/url)](/url2)` | `foo bar` | `foo [bar](/url)` | Link not stripped |
| 576 | `![foo *bar*][]` (ref) | `foo bar` | `foo *bar*` | Markup not stripped |
| 577 | `![foo *bar*][foobar]` (ref) | `foo bar` | `foo *bar*` | Markup not stripped |
| 585 | `![*foo* bar][]` (ref) | `foo bar` | `*foo* bar` | Markup not stripped |
| 589 | `![*foo* bar]` (ref) | `foo bar` | `*foo* bar` | Markup not stripped |

The CommonMark spec says image alt text is "the text content of the image
description" — meaning inline markup (emphasis, links, images) is resolved
and then stripped to plain text.

### Changes needed:
Two approaches:

**Option A (Parser-side):** Add a `stripInlineMarkup(text, refs)` function that
tokenizes inline content, resolves emphasis, and extracts only the text content
(no `*`, `_`, `[`, `]`, `(`, `)` from markup). Apply this in `parseInlineImage`
and `parseReferenceImage` to produce clean alt text.

**Option B (Renderer-side):** The renderer already receives `ev.Text` for images.
Change the renderer to process the alt text through inline parsing and extract
plain text. This is wrong — the renderer shouldn't re-parse.

**Option C (Hybrid):** The parser already calls `tokenizeLinkContent` for link
labels. For images, instead of using the raw label text as `ev.Text`, run the
label through inline tokenization and extract plain text.

**Go with Option A:** Add `plainTextFromInline(text string, refs map[string]linkReference) string` that:
1. Tokenizes inline content
2. Resolves emphasis
3. Concatenates only text content (skipping markup delimiters)
4. Recursively strips nested images/links

Apply in `parseInlineImage`, `parseInlineImageAsLink`, and `parseReferenceImage`.

### Files: `stream/parser_impl.go` (new function + image parse functions)

---

## Phase 4: Image Parsing — `!` Adjacent to `[` (1 fix) ⬜
**Example: 579**
**Difficulty: Easy | Impact: 1 example**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 579 | `My ![foo bar](/path/to/train.jpg  "title"   )` | `<img ...>` | `!<a ...>` | `!` not recognized as image prefix |

Looking at the GOT output: `My !<a href="/path/to/train.jpg" title="title">foo bar</a>` — the `!` is emitted as text and then `[foo bar](...)` is parsed as a link. This means `parseInlineImage` is failing.

The issue: `parseInlineImage` calls `parseInlineImageAsLink` which checks `strings.HasPrefix(text, "![")`. The input at the `!` position is `![foo bar](/path/to/train.jpg  "title"   )` — this should match. Let me check if the `imagePossible` guard is the problem... `imagePossible := strings.Contains(text, "](")` — the full text contains `](` so this should be true.

Wait — looking more carefully at the link title parsing. The destination has trailing spaces: `/path/to/train.jpg  "title"   )`. The `parseInlineLinkTail` needs to handle spaces between dest and title, and trailing spaces before `)`. Let me check `parseInlineLinkTail`:

### Changes needed:
Debug `parseInlineLinkTail` with input `/path/to/train.jpg  "title"   )` — likely the trailing spaces before `)` aren't handled.

### Files: `stream/parser_impl.go` (`parseInlineLinkTail`)

---

## Phase 5: Emphasis + Link Nesting (5 fixes) ⬜
**Examples: 404, 419, 422, 432, 433**
**Difficulty: Hard | Impact: 5 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 404 | `*foo [bar](/url)*` | `<em>foo <a>bar</a></em>` | `<em>foo <a>bar</em></a>` | Link closes after em |
| 419 | `*foo [*bar*](/url)*` | `<em>foo <a><em>bar</em></a></em>` | `<em>foo <a>bar</em></a>` | Inner em lost, nesting wrong |
| 422 | `**foo [bar](/url)**` | `<strong>foo <a>bar</a></strong>` | `<strong>foo <a>bar</strong></a>` | Link closes after strong |
| 432 | `**foo *bar **baz**\nbim* bop**` | Complex nesting | Wrong nesting | Multiline emphasis |
| 433 | `**foo [*bar*](/url)**` | `<strong>foo <a><em>bar</em></a></strong>` | `<strong>foo <a><em>bar</em></strong></a>` | Link closes after strong |

The core issue: when emphasis wraps a link (`*foo [bar](/url)*`), the emphasis
markers are outside the link. But `tokenizeLinkContent` processes the link label
independently, so the outer `*` opener doesn't see the inner content.

The current architecture: `tokenizeInline` encounters `[`, calls `parseInlineLink`
which extracts the label, then `tokenizeLinkContent` tokenizes the label with
the link style merged. The outer `*` delimiter is in the main token stream, but
the link content is already resolved — the `*` closer after `]` can't match
the opener before `[`.

### Changes needed:
This is the hardest fix. The fundamental issue is that links are parsed eagerly
and their content is flattened into the token stream with link styles applied.
The outer emphasis delimiters can't "see through" the link boundary.

**Approach:** Instead of eagerly resolving link content, emit the link as
boundary tokens in the main token stream:
1. Emit a "link open" token before the link content
2. Emit the link content tokens (with emphasis delimiters preserved)
3. Emit a "link close" token after
4. Run emphasis resolution on the full stream
5. In the output phase, apply link styles to tokens between open/close

This is a significant refactor of the inline tokenizer. Alternative: detect
the specific pattern where emphasis wraps a link and handle it specially.

### Files: `stream/parser_impl.go` (inline tokenizer, emphasis resolution)

---

## Phase 6: List Interruption Edge Cases (2 fixes) ⬜
**Examples: 367, 285**
**Difficulty: Medium | Impact: 2 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 367 | `*foo bar\n*` | `<p>*foo bar\n*</p>` | `*foo bar` + list | `*` at line start parsed as list |
| 285 | `foo\n*` / `foo\n1.` | Paragraph continuation | List interruption | Empty markers interrupt para |

### Changes needed:
1. **285**: An empty list marker (marker with no content after it) should not
   interrupt a paragraph. The `listItem` function returns success for `*\n` and
   `1.\n`, but the paragraph interruption check at line ~514-519 only blocks
   ordered lists with start != 1. Need to also block empty bullet items.
2. **367**: `*\n` at the end of a paragraph is being parsed as a list item
   instead of paragraph continuation. This is the same issue — `*` alone on a
   line shouldn't interrupt a paragraph.

### Files: `stream/parser_impl.go` (`processLine` paragraph interruption logic)

---

## Phase 7: Link Edge Cases (6 fixes) ⬜
**Examples: 505, 517, 526, 531, 533, 538**
**Difficulty: Medium-Hard | Impact: 6 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 505 | Three `[link](/url "title")` on separate lines | 3 separate links | One link spanning 3 lines | Newline in link label |
| 517 | `[![moon](moon.jpg)](/uri)` | `<a><img></a>` | `[<img>](/uri)` | Image inside link not parsed |
| 526 | `[foo<https://...](uri)>` | Autolink wins | Link wins | Autolink precedence |
| 531 | `[![moon](moon.jpg)][ref]` | `<a><img></a>` | `[<img>]<a>ref</a>` | Image inside ref link |
| 533 | `[foo *bar [baz][ref]*][ref]` | Specific nesting | Wrong | Ref link nesting |
| 538 | `[foo<https://...][ref]>` | Autolink wins | Link wins | Autolink precedence |

### Changes needed:
1. **505**: `matchingLinkLabelEnd` allows newlines in labels. The three lines
   form one big label `link](/url "title")\n[link](/url "title")\n[link](/url "title")`.
   Need to limit link labels to not span across what would be separate inline links.
   Actually — the issue is that the first `[` matches the last `]`, skipping
   the intermediate `](`s. `matchingBracketEnd` counts nesting but `](` inside
   should terminate the label. **Fix**: In `matchingLinkLabelEnd`, reject labels
   containing `](` (or limit label length to 999 chars per spec).
2. **517/531**: Image inside link — `[![moon](moon.jpg)](/uri)`. The outer `[`
   starts a link, the label contains `![moon](moon.jpg)` which is an image.
   `containsInlineLink` rejects this because it sees `[moon](moon.jpg)` as a
   nested link. **Fix**: `containsInlineLink` should not count image links
   (`![...](...)`) as nested links.
3. **526/538**: `<autolink>` inside `[...]` should take precedence. Currently
   the `[` is parsed first as a link opener. **Fix**: When scanning for `](`
   inside a link label, check if `<autolink>` spans across the `]` boundary.
4. **533**: Complex ref link nesting — the outer `[..][ref]` should fail because
   the inner `[baz][ref]` consumes the ref link.

### Files: `stream/parser_impl.go` (link parsing, `containsInlineLink`, `matchingBracketEnd`)

---

## Phase 8: Link Reference Definitions (5 fixes) ⬜
**Examples: 196, 208, 214, 218, 541**
**Difficulty: Medium-Hard | Impact: 5 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 196 | `[foo]: /url '\ntitle\n...\n'` | Multiline title | Not parsed | Multiline single-quoted title |
| 208 | `[\nfoo\n]: /url\nbar` | `<p>bar</p>` | All as paragraph | Multiline label |
| 214 | `# [Foo]\n[foo]: /url\n> bar` | Heading with link | `[Foo]` literal | Forward ref not resolved |
| 218 | `[foo]\n\n> [foo]: /url` | Link resolved | `[foo]` literal | Ref def in blockquote |
| 541 | `[Foo\n  bar]: /url\n\n[Baz][Foo bar]` | Link resolved | Not resolved | Multiline label in ref def |

### Changes needed:
1. **196**: `parseLinkReferenceDefinitionTail` / `continueLinkReferenceDefinition`
   doesn't support multiline titles. Need to buffer title lines and detect
   closing quote across lines.
2. **208/541**: `parseLinkReferenceDefinitionStart` doesn't support labels
   spanning multiple lines. Need to buffer the `[` and continue scanning on
   subsequent lines until `]:` is found.
3. **214**: The heading `# [Foo]` is emitted immediately with inline parsing,
   but `[foo]: /url` comes after. The pending blocks mechanism should defer
   the heading's inline parsing. Currently headings are parsed eagerly.
4. **218**: Ref defs inside blockquotes should be visible to content outside
   the blockquote (or at least to content before the blockquote that's in
   pending blocks). This requires the blockquote's ref defs to be collected
   before pending blocks are drained.

### Files: `stream/parser_impl.go` (ref def parsing, pending blocks, blockquote handling)

---

## Phase 9: Tabs — Partial Tab Expansion (3 fixes) ⬜
**Examples: 5, 6, 7**
**Difficulty: Medium | Impact: 3 examples**

| # | Input | Expected code content | Got | Root cause |
|---|-------|----------------------|-----|------------|
| 5 | `- foo\n\n\t\tbar` | `  bar` (2 spaces) | `bar` (0 spaces) | Tab not partially expanded |
| 6 | `>\t\tfoo` | `  foo` (2 spaces) | `foo` (0 spaces) | Tab not partially expanded |
| 7 | `-\t\tfoo` | `  foo` (2 spaces) | `foo` (0 spaces) | Tab not partially expanded |

After stripping the list marker (`- ` = 2 columns) or blockquote marker (`> ` = 2 columns),
the first tab expands to 2 spaces (filling to column 4), then the second tab is 4 spaces,
giving 6 columns of indent. Subtracting 4 for indented code leaves 2 spaces.

### Changes needed:
1. **`stripIndent`** and **`blockquoteContent`**: When stripping indent for
   container markers, track the column position and partially expand tabs.
   Currently `blockquoteContent` just strips `> ` literally without considering
   tab stops. Need column-aware stripping.

### Files: `stream/parser_impl.go` (`stripIndent`, `blockquoteContent`, `listItem`)

---

## Phase 10: Blockquote Continuation (3 fixes) ⬜
**Examples: 235, 250, 251**
**Difficulty: Hard | Impact: 3 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 235 | `> - foo\n- bar` | BQ(list(foo)) + list(bar) | BQ(list(foo, loose)) + list(bar) | Tight/loose wrong |
| 250 | `> > > foo\nbar` | Triple-nested BQ | Single BQ with literal `> >` | No nested BQ support |
| 251 | `>>> foo\n> bar\n>>baz` | Triple-nested BQ | Single BQ | No nested BQ support |

### Changes needed:
The parser currently supports only one level of blockquote (`inBlockquote` is a
single bool). Need a **blockquote stack** to support nested blockquotes.

1. Replace `inBlockquote bool` with a blockquote depth counter or stack.
2. `blockquoteContent` should be called recursively to strip multiple `>` prefixes.
3. Lazy continuation: `bar` after `> > > foo` continues the innermost blockquote.
4. **235**: The list inside the blockquote has only one item (`foo`), so it should
   be tight. The `- bar` is outside the blockquote.

### Files: `stream/parser_impl.go` (blockquote handling — major refactor)

---

## Phase 11: HTML Blocks (4 fixes) ⬜
**Examples: 148, 168, 174, 175**
**Difficulty: Medium | Impact: 4 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 148 | `<table><tr><td>\n<pre>\n**Hello**,\n\n_world_.\n</pre>` | Mixed HTML+MD | All escaped | Type 1 HTML block not recognized |
| 168 | `<del>*foo*</del>` | `<p><del><em>foo</em></del></p>` | Escaped `<del>` | `<del>` treated as type 6 block |
| 174 | `> <div>\n> foo\n\nbar` | BQ(div+foo) + p(bar) | BQ(escaped div) | HTML block in BQ |
| 175 | `- <div>\n- foo` | list(div, foo) | list(escaped div + `- foo`) | HTML block in list |

### Changes needed:
1. **168**: `<del>` is not a block-level tag (not in the type 6 list). It should
   be parsed as inline raw HTML, not an HTML block. Check if `del` is in
   `htmlBlockType6Tags`. If it is, remove it — `del` is an inline element.
2. **148**: `<table><tr><td>` is type 6. The blank line between `**Hello**,` and
   `_world_.` should end the HTML block and start a new paragraph. Type 6 HTML
   blocks end at a blank line.
3. **174/175**: HTML blocks inside blockquotes and list items. The `>` prefix
   stripping and list item content routing need to detect HTML block starts.

### Files: `stream/parser_impl.go` (HTML block detection, `htmlBlockType6Tags`, container routing)

---

## Phase 12: List Items — Container Nesting (4 fixes) ⬜
**Examples: 259, 260, 280, 292, 293**
**Difficulty: Hard | Impact: 5 examples**
*Note: 285 covered in Phase 6*

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 259 | `   > > 1.  one\n>>\n>>     two` | Nested BQ+OL | Single BQ | No nested BQ |
| 260 | `>>- one\n>>\n  >  > two` | Nested BQ+UL+p | Single BQ | No nested BQ |
| 280 | `-\n\n  foo` | Empty item + p(foo) | Item with p(foo) | Empty item continuation |
| 292 | `> 1. > Blockquote\ncontinued here.` | BQ>OL>BQ>p | BQ>OL>p | BQ inside list item |
| 293 | `> 1. > Blockquote\n> continued here.` | BQ>OL>BQ>p | BQ>OL>p | BQ inside list item |

### Changes needed:
1. **280**: `-\n\n  foo` — the `-` creates an empty list item. The blank line
   should close it. `  foo` (2 spaces) matches the list item indent (2), but
   the item was empty and closed by blank line, so it shouldn't continue.
   **Fix**: Track whether a list item ever had content; if not, blank line
   closes it permanently.
2. **292/293**: Blockquote inside list item inside blockquote. Requires nested
   blockquote support (Phase 10).
3. **259/260**: Deeply nested blockquote+list. Requires nested blockquote support.

### Files: `stream/parser_impl.go` (list item handling, blockquote nesting)

---

## Phase 13: Lists — Tight/Loose + Nesting (3 fixes) ⬜
**Examples: 319, 325, 326**
**Difficulty: Medium | Impact: 3 examples**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 319 | `- a\n  - b\n\n    c\n- d` | Outer tight, inner loose | `d` in inner list | Sublist boundary wrong |
| 325 | `* foo\n  * bar\n\n  baz` | Outer loose, inner tight | Inner loose | Blank line attribution |
| 326 | `- a\n  - b\n  - c\n\n- d` | Outer loose, inner tight | Outer tight, inner loose | Blank line attribution |

### Changes needed:
The blank line between items needs to be attributed to the correct list level.
A blank line followed by content at the *outer* list's indent makes the *outer*
list loose, not the inner one.

1. **319**: `    c` (4 spaces) is at the inner list's indent (4), so it continues
   the inner list item `b`. `- d` is at the outer list's indent (0), so it's a
   new outer item. The inner list is loose (blank line before `c`), outer is tight.
2. **325**: `  baz` (2 spaces) is at the outer list's indent (2), not the inner
   list's (4). So `baz` is a continuation of the outer item, making the outer
   list loose. The inner list has no blank lines → tight.
3. **326**: The blank line between `- c` and `- d` is between outer items →
   outer list is loose. Inner lists have no blank lines → tight.

**Fix**: When a blank line is followed by content, check which list level the
content belongs to. Only mark that level's list as loose.

### Files: `stream/parser_impl.go` (list continuation, tight/loose logic)

---

## Phase 14: Fenced Code in Blockquote (1 fix) ⬜
**Example: 128**
**Difficulty: Easy | Impact: 1 example**

| # | Input | Expected | Got | Root cause |
|---|-------|----------|-----|------------|
| 128 | `> \`\`\`\n> aaa\n\nbbb` | BQ(code(aaa)) + p(bbb) | BQ(code(> aaa)) | `>` prefix not stripped from fence content |

### Changes needed:
When a fenced code block is inside a blockquote, the `>` prefix is being
included in the code content. The blockquote content routing at line ~442-452
strips the `>` prefix but the content still has it.

**Fix**: Check the blockquote fence content path — `inner.text` should already
have the `>` stripped by `blockquoteContent`. Debug to find where the prefix
leaks through.

### Files: `stream/parser_impl.go` (blockquote + fence interaction)

---

## Recommended Execution Order

Priority by difficulty and impact:

| Order | Phase | Examples | Difficulty | Impact |
|-------|-------|----------|------------|--------|
| 1 | **Phase 2**: Raw HTML comment | 626 | Easy | 1 |
| 2 | **Phase 1**: Autolinks | 602,603,608,611,612 | Easy | 5 |
| 3 | **Phase 6**: List interruption | 367, 285 | Easy-Med | 2 |
| 4 | **Phase 4**: Image `!` parsing | 579 | Easy | 1 |
| 5 | **Phase 3**: Image alt text | 573-577,585,589 | Medium | 7 |
| 6 | **Phase 14**: Fence in blockquote | 128 | Easy | 1 |
| 7 | **Phase 11**: HTML blocks | 148,168,174,175 | Medium | 4 |
| 8 | **Phase 7**: Link edge cases | 505,517,526,531,533,538 | Med-Hard | 6 |
| 9 | **Phase 8**: Link ref defs | 196,208,214,218,541 | Med-Hard | 5 |
| 10 | **Phase 13**: List tight/loose | 319,325,326 | Medium | 3 |
| 11 | **Phase 12**: List container nesting | 259,260,280,292,293 | Hard | 5 |
| 12 | **Phase 9**: Tabs | 5,6,7 | Medium | 3 |
| 13 | **Phase 5**: Emphasis+link nesting | 404,419,422,432,433 | Hard | 5 |
| 14 | **Phase 10**: Nested blockquotes | 235,250,251 | Hard | 3 |

**Easy wins (phases 1-4,6,14):** 17 examples → 617/652 (94.6%)
**Medium work (phases 3,7,8,11,13):** 25 examples → 642/652 (98.5%)
**Hard work (phases 5,9,10,12):** 16 examples → 652/652 (100%)

---

## Cross-cutting Concerns

### Nested Blockquotes (affects Phases 10, 12, 14)
The single `inBlockquote bool` is the biggest architectural limitation. Fixing
examples 250, 251, 259, 260, 292, 293 all require a blockquote stack. This is
a ~200-line refactor touching `processLine`, `closeBlockquote`, `blockquoteContent`,
and all container interaction code.

### Emphasis + Link Architecture (Phase 5)
The current eager link resolution in `tokenizeInline` prevents emphasis from
spanning across link boundaries. Fixing this requires changing links from
eagerly-resolved tokens to boundary markers in the token stream, then resolving
emphasis across the full stream. This is the deepest architectural change.

### Test Strategy
Each phase should:
1. Add targeted unit tests in `stream/parser_test.go`
2. Run `TestHTMLCommonMarkCompliance` to verify example count increases
3. Run full test suite to check for regressions
