# mdview

Terminal Markdown viewer powered by [codewandler/markdown](https://github.com/codewandler/markdown).

```bash
mdview README.md
echo "**hello**" | mdview
cat doc.md | mdview --width 80
```

## Install

```bash
go install github.com/codewandler/markdown/cmd/mdview@latest
```

## Usage

```
Usage: mdview [flags] [file]

Render Markdown to the terminal.

If no file is given, reads from stdin.

Flags:
      --chunk int                   bytes per streaming chunk when --stream is set (default 16)
      --delay duration              delay between chunks when --stream is set (default 20ms)
      --live                        use live renderer with redrawable tables when stdout is a terminal
      --no-color                    disable ANSI colors
      --no-wrap                     disable word wrapping
      --stream                      render markdown in delayed chunks for testing streaming behavior
      --table-max-width int         maximum table width for auto mode (0 = wrap width or terminal width)
      --table-mode string           table rendering mode: buffered, fixed, or auto (default "buffered")
      --table-overflow string       fixed/auto table overflow: ellipsis or clip (default "ellipsis")
      --table-widths string         comma-separated fixed table column widths, e.g. 16,12,40
      --version                     print version and exit
      --width int                   wrap width (0 = auto-detect terminal)
```

## Features

- Streaming output — starts rendering before the file is fully read
- Syntax highlighting — Go via stdlib AST fast path, others via Chroma
- OSC 8 clickable hyperlinks
- Word wrapping with auto-detected terminal width
- Buffered, fixed-width, auto-width, and live table rendering modes
- Emoji shortcode rendering via inline scanner extensions
- Terminal image rendering for supported terminals
- TTY detection — ANSI stripped when piped

## Roadmap

- [x] **Emoji shortcodes** — `:white_check_mark:` → ✅, `:rocket:` → 🚀
- [x] **Terminal images** — render `![alt](image.png)` with Kitty graphics protocol where supported
- [x] **HTML tag stripping** — clean up wrapper tags and badge noise in output
- [ ] **Scrollable viewport** — [Bubble Tea](https://github.com/charmbracelet/bubbletea)
      pager with `j`/`k`/arrows, `q` to quit, `/` to search
- [ ] **`--pager`/`--no-pager`** — auto-detect when output exceeds terminal height
- [ ] **`--theme`** — color theme selection (monokai, dracula, nord)
