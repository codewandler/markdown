package terminal

// styler abstracts ANSI escape emission so the renderer can operate in
// both colour (TTY) and plain (non-TTY) modes without conditional checks
// scattered across every call site.
type styler interface {
	// Named colours
	comment(s string) string
	fg(s string) string
	green(s string) string
	yellow(s string) string
	blue(s string) string

	// Text attributes
	bold(s string) string
	italic(s string) string
	strike(s string) string
	underline(s string) string
	reset() string

	// Hyperlinks
	link(url, text string) string

	// Raw escape — for cases where a raw ANSI string is needed (e.g. BorderColor)
	// Returns the code itself when colour, empty string when plain.
	code(ansi string) string
}

// ── ANSI styler ───────────────────────────────────────────────────────────────

type ansiStyler struct{}

func (ansiStyler) comment(s string) string   { return monokaiComment + s + reset }
func (ansiStyler) fg(s string) string        { return monokaiForeground + s + reset }
func (ansiStyler) green(s string) string     { return monokaiGreen + s + reset }
func (ansiStyler) yellow(s string) string    { return monokaiYellow + s + reset }
func (ansiStyler) blue(s string) string      { return monokaiBlue + s + reset }
func (ansiStyler) bold(s string) string      { return bold + s + reset }
func (ansiStyler) italic(s string) string    { return italic + s + reset }
func (ansiStyler) strike(s string) string    { return strike + s + reset }
func (ansiStyler) underline(s string) string { return underline + s + reset }
func (ansiStyler) reset() string             { return reset }
func (ansiStyler) code(ansi string) string   { return ansi }
func (ansiStyler) link(url, text string) string {
	return underline + monokaiBlue + osc8Open(url) + text + osc8Close() + reset + monokaiForeground
}

// ── Plain styler ──────────────────────────────────────────────────────────────

type plainStyler struct{}

func (plainStyler) comment(s string) string   { return s }
func (plainStyler) fg(s string) string        { return s }
func (plainStyler) green(s string) string     { return s }
func (plainStyler) yellow(s string) string    { return s }
func (plainStyler) blue(s string) string      { return s }
func (plainStyler) bold(s string) string      { return s }
func (plainStyler) italic(s string) string    { return s }
func (plainStyler) strike(s string) string    { return s }
func (plainStyler) underline(s string) string { return s }
func (plainStyler) reset() string             { return "" }
func (plainStyler) code(ansi string) string   { return "" }
func (plainStyler) link(url, text string) string { return text }
