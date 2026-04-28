# CommonMark Gap Audit

**Date:** 2026-04-28
**Corpus:** CommonMark 0.31.2 (652 examples)
**Baseline:** v0.26.0 — 307 supported, 174 known gap, 171 unsupported

## Summary by Root Cause

| Root Cause | Examples | Sections Affected |
|---|---|---|
| **Emphasis algorithm** | ~107 | Emphasis and strong emphasis |
| **Reference-style links** | ~71 | Links, Images, Link ref defs, Backslash, Entities |
| **Container stack** | ~68 | List items, Lists, Block quotes, Tabs, Fenced code, Thematic breaks, Indented code |
| **HTML blocks + raw HTML** | ~64 | HTML blocks (44), Raw HTML (20) |
| **Link/emphasis/code interaction** | ~15 | Links (inline precedence) |
| **Misc edge cases** | ~13 | Fenced code, Code spans, Entities |

## Root Cause 1: Emphasis Algorithm (107 examples)

**Section:** Emphasis and strong emphasis (examples 350–481)
**Currently supported:** 25 of 132

The current `resolveEmphasis` uses a simple greedy stack: scan for closing
delimiters, match the nearest open delimiter of the same marker. This misses:

- **Rule of three:** when both open and close delimiters can be both opening
  and closing, and the sum of their lengths is a multiple of 3, they must not
  match unless both are multiples of 3. (spec rules 9–10)
- **Multiple-of-three priority:** the algorithm must skip candidate openers
  that violate the rule-of-three constraint.
- **Proper `*` vs `_` flanking:** `_` has additional restrictions (can't open
  if right-flanking unless preceded by punctuation, etc.). The current code
  handles this in `emphasisDelimRun` but the matching algorithm doesn't
  enforce the constraint during pairing.
- **Nested/interleaved emphasis:** `*foo **bar** baz*` must produce
  `<em>foo <strong>bar</strong> baz</em>`. The current greedy stack sometimes
  pairs wrong delimiters when runs have different lengths.
- **Partial consumption:** a `***` run should be split into `*` + `**` (or
  `**` + `*`) depending on what it matches. The current code handles `use`
  counts but doesn't re-push remainders onto the stack.

**Fix:** Replace `resolveEmphasis` with the CommonMark "process emphasis"
algorithm (spec appendix). This is a well-defined algorithm with an opener
stack, closer scanning, rule-of-three checks, and partial consumption.

## Root Cause 2: Reference-Style Links (~71 examples)

**Sections:** Links (60 gaps), Images (15 gaps), Link ref defs (9 gaps),
Backslash escapes (2), Entities (2)

The parser already has a `refs` map and resolves `[foo]` shortcut references
and `[text](url)` direct links. Missing:

- **Full-reference links:** `[foo][bar]` — look up `bar` in refs map.
- **Collapsed-reference links:** `[foo][]` — look up `foo` in refs map.
- **Shortcut-reference links:** `[foo]` — already partially works but needs
  case-insensitive matching and Unicode case folding.
- **Case-insensitive label matching:** `[Foo]` should match `[foo]: /url`.
  Spec requires Unicode case fold, not just ASCII toLower.
- **Forward references:** `[foo]` appears before `[foo]: /url`. The
  append-only streaming model makes this inherently hard. Current approach:
  emit paragraph text, can't retroactively turn it into a link. This is an
  acceptable known gap for the streaming model.
- **Reference images:** `![foo][bar]`, `![foo][]`, `![foo]` — same resolution
  logic as links but producing image events.
- **Multiline labels:** `[Foo\n  bar]: /url` — label spans multiple lines.
- **Escaped characters in labels/destinations:** `[foo\\]`, `[bar*]` etc.

**Fix:** Implement full/collapsed/shortcut reference resolution in the inline
parser. Add case-insensitive label lookup. Forward references remain a known
gap (acceptable for streaming).

## Root Cause 3: Container Stack (~68 examples)

**Sections:** List items (34), Lists (21), Block quotes (6), Tabs (5),
Fenced code (2), Thematic breaks (1), Indented code (2)

The parser uses flat boolean flags (`inBlockquote`, `inList`, `inListItem`)
instead of a real container stack. This means:

- **No nested lists:** `- foo\n  - bar` can't produce a sub-list inside a
  list item. The parser treats indented content as paragraph continuation.
- **No blocks inside list items:** list items can't contain fenced code,
  blockquotes, indented code, or sub-lists.
- **No nested blockquotes:** `> > > foo` can't produce triple-nested
  blockquotes.
- **No loose/tight detection:** a list is "loose" if any item is separated by
  blank lines. Loose items wrap content in `<p>`. The parser doesn't track
  this.
- **Indentation-based continuation:** list item continuation requires
  tracking the column width of the list marker + padding, then treating lines
  indented past that as belonging to the item.

**Fix:** Replace the flat flags with a container stack (slice of container
frames). Each frame tracks its type (document, blockquote, list, list_item),
indentation, and state. On each line, walk the stack to determine which
containers continue, which close, and whether new containers open. This is
the largest single change in the plan.

## Root Cause 4: HTML Blocks + Raw HTML (64 examples)

**Sections:** HTML blocks (44), Raw HTML (20)

Currently completely unsupported. The parser has a `BlockHTML` block kind
defined but no recognition logic.

- **HTML blocks (types 1–7):** recognized by opening patterns like `<pre`,
  `<script`, `<div`, `<!--`, `<?`, `<!`, `<tag>`, or any block-level tag
  alone on a line. Each type has different closing conditions.
- **Raw HTML inlines:** `<a href="...">`, `</div>`, `<!-- comment -->` etc.
  inside paragraph text. These should pass through as literal text events
  (the terminal renderer will strip or ignore them).

**Fix:** Add HTML block recognition in `processLine` (before paragraph
fallback). Add raw HTML inline recognition in the inline tokenizer. The
AGENTS.md says "HTML rendering is out of scope unless explicitly added as a
real incremental renderer" — but recognizing HTML *blocks* as blocks (and
passing content through) is different from rendering HTML. The parser should
recognize them to avoid misinterpreting HTML as Markdown.

## Root Cause 5: Link/Emphasis/Code Interaction (15 examples)

**Section:** Links (examples 513–526)

These test precedence rules when links, emphasis, code spans, and raw HTML
interact:

- Code spans take precedence over links: `` [foo`](/uri)` `` → the backtick
  opens a code span that swallows the `](/uri)`.
- Links take precedence over emphasis: `*[foo*](/uri)` → the `[` opens a
  link that contains `foo*`, emphasis doesn't match across the link boundary.
- Nested links are not allowed: `[foo [bar](/uri)](/uri)` → inner link wins.
- Images inside links: `[![moon](moon.jpg)](/uri)` → image inside link.

**Fix:** Most of these will be resolved by implementing the CommonMark
emphasis algorithm (which respects delimiter boundaries) and by making the
inline parser handle link/code-span precedence correctly. The inline
tokenizer already handles code spans before emphasis; the remaining gaps are
about link bracket matching interacting with emphasis boundaries.

## Root Cause 6: Misc Edge Cases (13 examples)

Scattered across sections:

- **Fenced code #121:** 2-backtick fence (`` `` ``) is a code span, not a
  fence. Parser incorrectly treats it as a fence.
- **Fenced code #127, #139:** closing fence with trailing content, or
  non-matching close.
- **Fenced code #128:** fenced code inside blockquote with lazy continuation.
- **Fenced code #138, #145:** backtick fences with backticks in info string.
- **Fenced code #141:** setext heading + fenced code interaction.
- **Fenced code #146:** tilde fence with backticks in info string (allowed).
- **Code spans #341, #342:** emphasis/link delimiters inside code spans.
- **Entities #32, #41:** character references in link destinations/titles.
- **Thematic break #59:** `Foo\n---\nbar` is setext heading, not thematic
  break + paragraph. (May already work — needs verification.)

## Priority Order

1. **Emphasis algorithm** — largest single block (107 examples), self-contained
   change to `resolveEmphasis`, no structural parser changes needed.
2. **Reference links** — second largest (71 examples), extends existing inline
   parser, no structural changes to block parsing.
3. **Container stack** — hardest change (68 examples), requires rearchitecting
   the block parser's state model. But unlocks lists, nested blockquotes, and
   many cross-cutting edge cases.
4. **HTML blocks + raw HTML** — 64 examples, moderate complexity, mostly
   additive (new recognition patterns).
5. **Link/emphasis/code interaction** — 15 examples, mostly falls out of
   fixing #1 and #2.
6. **Misc edge cases** — 13 examples, individual fixes.

## Target After Full Implementation

If all root causes are addressed:

- Emphasis: 25 → ~120 supported (some edge cases may remain known gaps)
- Links: 30 → ~80 supported
- Images: 7 ��� ~20 supported
- List items: 14 → ~40 supported
- Lists: 5 → ~22 supported
- HTML blocks: 0 → ~40 supported
- Raw HTML: 0 → ~18 supported
- Other sections: close most remaining gaps

**Estimated total: 307 → ~500+ supported (77%+), with known gaps primarily
in forward references and deeply nested container edge cases.**
