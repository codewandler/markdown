# Go Markdown Libraries

A survey of Go libraries for Markdown parsing and terminal rendering,
current as of April 2026.

## Terminal Renderers

### glamour (charmbracelet)

- **Repo**: https://github.com/charmbracelet/glamour
- **Stars**: ~4.5K
- **Parser**: goldmark (CommonMark compliant)
- **Approach**: Batch — parse full document, render to styled string
- **Streaming**: No (re-renders entire document on each update)
- **Syntax highlighting**: Chroma with customizable themes
- **Tables**: Yes (redesigned in v0.10.0 with screen reader support)
- **Task lists**: Yes (via goldmark GFM extension)
- **Strikethrough**: Yes (via goldmark GFM extension)
- **Links**: Rendered as text + URL, no OSC 8 hyperlinks
- **Word wrapping**: Fixed width, configurable
- **TTY detection**: No (manual style selection)
- **Themes**: Multiple built-in (dark, light, dracula, tokyo-night, etc.)
- **Dependencies**: goldmark, lipgloss, bluemonday, reflow, termenv (~20 transitive)
- **Used by**: glow, streamd, GitHub CLI

### go-term-markdown (MichaelMure)

- **Repo**: https://github.com/MichaelMure/go-term-markdown
- **Stars**: ~100
- **Parser**: blackfriday v1 (not CommonMark compliant)
- **Approach**: Batch — parse full document, render to string
- **Streaming**: No
- **Syntax highlighting**: Chroma v1 (older API)
- **Tables**: Yes
- **Task lists**: No
- **Strikethrough**: No
- **Links**: Rendered as text, no hyperlinks
- **Word wrapping**: Fixed width parameter
- **TTY detection**: No
- **Images**: Inline terminal images via pixterm (unique feature)
- **Dependencies**: blackfriday, chroma v1, go-term-text, pixterm (~15 transitive)
- **Used by**: git-bug

### streamd (Gaurav-Gosain)

- **Repo**: https://github.com/Gaurav-Gosain/streamd
- **Stars**: ~2
- **Parser**: glamour (which uses goldmark)
- **Approach**: CLI tool — re-renders via glamour on each chunk
- **Streaming**: Pseudo-streaming (re-renders full document, not append-only)
- **Note**: CLI tool, not a library. Uses DECSYNC for flicker-free re-rendering.
- **Used by**: standalone CLI

### codewandler/markdown (this library)

- **Repo**: https://github.com/codewandler/markdown
- **Parser**: Custom streaming parser (no goldmark/blackfriday dependency)
- **Approach**: True streaming — append-only events, chunk-safe
- **Streaming**: Yes (only Go library with true streaming)
- **Syntax highlighting**: Go stdlib AST fast path + Chroma for other languages
- **Tables**: Yes (GFM, with alignment)
- **Task lists**: Yes (GFM)
- **Strikethrough**: Yes (GFM)
- **Links**: OSC 8 clickable terminal hyperlinks
- **Word wrapping**: Auto-detected terminal width, configurable
- **TTY detection**: Auto (strips ANSI when piped)
- **CommonMark**: 96.2% (627/652)
- **GFM**: 100% (672/672)
- **Dependencies**: chroma + regexp2 (2 direct deps)

## Parse-Only Libraries

### goldmark (yuin)

- **Repo**: https://github.com/yuin/goldmark
- **Stars**: ~3.8K
- **Approach**: Batch parser, produces AST
- **CommonMark**: 100% compliant
- **GFM**: Via extensions (tables, strikethrough, task lists, autolinks)
- **Streaming**: No
- **Output**: AST (renderers are separate)
- **Dependencies**: Zero (pure Go stdlib)
- **Note**: The de facto standard Go Markdown parser. Used by glamour,
  Hugo, and many others.

### blackfriday v2 (russross)

- **Repo**: https://github.com/russross/blackfriday
- **Stars**: ~5.5K
- **Approach**: Batch parser + HTML renderer
- **CommonMark**: Not compliant (custom dialect)
- **Streaming**: No
- **Output**: HTML string
- **Dependencies**: Zero
- **Note**: Fast but unmaintained since 2019. Not CommonMark compliant.
  Still widely used in legacy projects.

### gomarkdown/markdown

- **Repo**: https://github.com/gomarkdown/markdown
- **Stars**: ~1.4K
- **Approach**: Batch parser + HTML renderer
- **CommonMark**: Partial compliance
- **Streaming**: No
- **Output**: HTML string
- **Dependencies**: Zero
- **Note**: Active fork of blackfriday with ongoing maintenance.
  Supports many extensions but not fully CommonMark compliant.

## Non-Go Streaming Renderers (for reference)

| Library | Language | Streaming | Notes |
| --- | --- | --- | --- |
| mdstream | Rust | Yes | Streaming middleware for LLM output |
| md2term | Python | Yes | Terminal renderer using Rich |
| streaming-markdown | TypeScript | Yes | Browser-based streaming renderer |
| incremark | JavaScript | Yes | Incremental parser for AI streaming |

None of these are usable from Go without FFI/CGo.
