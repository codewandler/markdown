// Command mdview renders Markdown to the terminal.
//
// Usage:
//
//	mdview [flags] [file]
//	echo "**hello**" | mdview
//	cat README.md | mdview --width 80
//
// If no file is given, mdview reads from stdin.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/codewandler/markdown/bubbleview"
	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	width         int
	noWrap        bool
	noColor       bool
	tableMode     string
	tableWidths   string
	tableOverflow string
	tableMaxWidth int
	theme         string
	fileLinks     bool
	streamInput   bool
	chunk         int
	delay         time.Duration
	live          bool
	pager         bool
	showVersion   bool
}

func main() {
	cmd := newRootCommand(os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cfg := cliOptions{
		tableMode:     "buffered",
		tableOverflow: "ellipsis",
		theme:         "monokai",
		fileLinks:     true,
		chunk:         16,
		delay:         20 * time.Millisecond,
	}
	cmd := &cobra.Command{
		Use:           "mdview [file]",
		Short:         "Render Markdown to the terminal",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.showVersion {
				fmt.Fprintln(stdout, versionString())
				return nil
			}
			return run(cfg, args, stdout, stderr)
		},
	}
	cmd.SetOut(stderr)
	cmd.SetErr(stderr)
	cmd.Flags().IntVar(&cfg.width, "width", 0, "wrap width (0 = auto-detect terminal)")
	cmd.Flags().BoolVar(&cfg.noWrap, "no-wrap", false, "disable word wrapping")
	cmd.Flags().BoolVar(&cfg.noColor, "no-color", false, "disable ANSI colors")
	cmd.Flags().StringVar(&cfg.tableMode, "table-mode", cfg.tableMode, "table rendering mode: buffered, fixed, or auto")
	cmd.Flags().StringVar(&cfg.tableWidths, "table-widths", "", "comma-separated fixed table column widths, e.g. 16,12,40")
	cmd.Flags().StringVar(&cfg.tableOverflow, "table-overflow", cfg.tableOverflow, "fixed/auto table overflow: ellipsis or clip")
	cmd.Flags().IntVar(&cfg.tableMaxWidth, "table-max-width", 0, "maximum table width for auto mode (0 = wrap width or terminal width)")
	cmd.Flags().StringVar(&cfg.theme, "theme", cfg.theme, "terminal theme: monokai, nord, or plain")
	cmd.Flags().BoolVar(&cfg.fileLinks, "file-links", cfg.fileLinks, "render file references like foo.go:18 as OSC 8 file links")
	cmd.Flags().BoolVar(&cfg.streamInput, "stream", false, "render markdown in delayed chunks for testing streaming behavior")
	cmd.Flags().IntVar(&cfg.chunk, "chunk", cfg.chunk, "bytes per streaming chunk when --stream is set")
	cmd.Flags().DurationVar(&cfg.delay, "delay", cfg.delay, "delay between chunks when --stream is set")
	cmd.Flags().BoolVar(&cfg.live, "live", false, "use live renderer with redrawable tables when stdout is a terminal")
	cmd.Flags().BoolVar(&cfg.pager, "pager", false, "open rendered Markdown in an interactive terminal pager")
	cmd.Flags().BoolVar(&cfg.showVersion, "version", false, "print version and exit")
	return cmd
}

func run(cfg cliOptions, args []string, stdout, stderr io.Writer) error {
	var r io.Reader
	if len(args) > 0 {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	} else {
		// Check if stdin is a terminal (no input piped).
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice != 0 {
			return fmt.Errorf("Usage: mdview [file] or pipe input")
		}
		r = os.Stdin
	}

	var opts []terminal.RendererOption
	if cfg.width < 0 {
		return fmt.Errorf("--width must be >= 0")
	}
	if cfg.noWrap && cfg.width > 0 {
		return fmt.Errorf("--no-wrap cannot be used with --width")
	}
	if cfg.noWrap {
		opts = append(opts, terminal.WithWrapWidth(0))
	} else if cfg.width > 0 {
		opts = append(opts, terminal.WithWrapWidth(cfg.width))
	}
	if cfg.streamInput && cfg.chunk <= 0 {
		return fmt.Errorf("--chunk must be greater than zero")
	}
	theme, err := parseTheme(cfg.theme)
	if err != nil {
		return err
	}
	opts = append(opts, terminal.WithTheme(theme))
	if cfg.noColor {
		opts = append(opts, terminal.WithAnsi(terminal.AnsiOff))
	}
	tableLayout, err := parseTableLayout(cfg.tableMode, cfg.tableWidths, cfg.tableOverflow, cfg.tableMaxWidth)
	if err != nil {
		return err
	}
	if tableLayout.Mode != terminal.TableModeBuffered {
		opts = append(opts, terminal.WithTableLayout(tableLayout))
	}

	// Read all input.
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	input := string(raw)

	// Preprocess: strip HTML noise (<div>, </div>, badges, etc).
	input = stripHTML(input)

	// Determine base directory for resolving relative image paths.
	var baseDir string
	if len(args) > 0 {
		baseDir = filepath.Dir(args[0])
	} else {
		baseDir, _ = os.Getwd()
	}
	opts = append(opts,
		terminal.WithParserOptions(
			stream.WithInlineScanner(emojiScanner{}),
			stream.WithInlineScanner(fileRefScanner{}),
		),
		terminal.WithInlineRenderer("file-ref", fileRefRenderer(baseDir, cfg.fileLinks)),
	)

	if cfg.pager {
		if cfg.live {
			return fmt.Errorf("--pager cannot be used with --live")
		}
		if cfg.streamInput {
			return fmt.Errorf("--pager cannot be used with --stream")
		}
		return runPager([]byte(input), opts...)
	}

	// Split input into segments: markdown text and image placeholders.
	// Images are rendered directly to stdout, bypassing the Markdown parser.
	segments := splitImages(input, baseDir)
	liveEnabled, liveFallback := liveMode(cfg.live, stdout)
	if liveFallback {
		fmt.Fprintln(stderr, "mdview: --live requested but stdout is not a terminal; using append-only rendering")
	}
	sr := newMarkdownRenderer(stdout, liveEnabled, opts...)
	for _, seg := range segments {
		if seg.isImage {
			// Flush any pending Markdown before the image.
			if err := sr.Flush(); err != nil {
				return err
			}
			// Reset the renderer for the next segment.
			sr = newMarkdownRenderer(stdout, liveEnabled, opts...)
			// Write image directly to stdout.
			fmt.Fprint(stdout, seg.content)
		} else {
			if err := writeMarkdownSegment(sr, seg.content, cfg.streamInput, cfg.chunk, cfg.delay); err != nil {
				return err
			}
		}
	}
	if err := sr.Flush(); err != nil {
		return err
	}
	return nil
}

func runPager(input []byte, opts ...terminal.RendererOption) error {
	modelOpts := make([]bubbleview.Option, 0, len(opts))
	for _, opt := range opts {
		if opt != nil {
			modelOpts = append(modelOpts, bubbleview.WithRendererOption(opt))
		}
	}
	program := tea.NewProgram(bubbleview.NewPagerModel(input, modelOpts...), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newMarkdownRenderer(w io.Writer, live bool, opts ...terminal.RendererOption) markdownRenderer {
	if live {
		return terminal.NewLiveRenderer(w, opts...)
	}
	return terminal.NewStreamRenderer(w, opts...)
}

func liveMode(requested bool, w io.Writer) (enabled bool, fallback bool) {
	if !requested {
		return false, false
	}
	if !isTerminalWriter(w) {
		return false, true
	}
	return true, false
}

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

type markdownRenderer interface {
	Write([]byte) (int, error)
	Flush() error
}

func writeMarkdownSegment(sr markdownRenderer, content string, stream bool, chunk int, delay time.Duration) error {
	if !stream {
		_, err := sr.Write([]byte(content))
		return err
	}
	input := []byte(content)
	for len(input) > 0 {
		n := chunk
		if n > len(input) {
			n = len(input)
		}
		if _, err := sr.Write(input[:n]); err != nil {
			return err
		}
		input = input[n:]
		if delay > 0 && len(input) > 0 {
			time.Sleep(delay)
		}
	}
	return nil
}
