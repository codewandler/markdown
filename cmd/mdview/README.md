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
  -width int     wrap width (0 = auto-detect terminal)
  -no-color      disable ANSI colors
```

## Features

- Streaming output — starts rendering before the file is fully read
- Syntax highlighting — Go via stdlib AST fast path, others via Chroma
- OSC 8 clickable hyperlinks
- Word wrapping with auto-detected terminal width
- TTY detection — ANSI stripped when piped

## Roadmap

- [ ] **Emoji shortcodes** — `:white_check_mark:` → ✅, `:rocket:` → 🚀
- [ ] **Terminal images** — render `![alt](image.png)` as ANSI pixel art
      via [pixterm](https://github.com/eliukblau/pixterm) ansimage
- [ ] **HTML tag stripping** — clean up `<div>`, `<img>` etc. in output
- [ ] **Scrollable viewport** — [Bubble Tea](https://github.com/charmbracelet/bubbletea)
      pager with `j`/`k`/arrows, `q` to quit, `/` to search
- [ ] **`--pager`/`--no-pager`** — auto-detect when output exceeds terminal height
- [ ] **`--theme`** — color theme selection (monokai, dracula, nord)
