package competition

import (
	"io"

	goterm "github.com/MichaelMure/go-term-markdown"
	"github.com/charmbracelet/glamour"
	"github.com/codewandler/markdown"
	"github.com/codewandler/markdown/html"
	"github.com/codewandler/markdown/terminal"
	gomarkdown "github.com/gomarkdown/markdown"
	gomarkdownhtml "github.com/gomarkdown/markdown/html"
	gomarkdownparser "github.com/gomarkdown/markdown/parser"
	blackfriday "github.com/russross/blackfriday/v2"
	goldmark "github.com/yuin/goldmark"
	goldmarkext "github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	goldmarktext "github.com/yuin/goldmark/text"
)

// All is the complete list of candidates for the competition pipeline.
var All = []Candidate{
	{
		Repo: "https://github.com/codewandler/markdown",
		Features: Features{
			Parser:             "custom streaming",
			TerminalRender:     true,
			Streaming:          true,
			SyntaxHighlighting: "Go fast path + Chroma",
			ClickableLinks:     true,
			WordWrap:           "auto-detect",
			TTYDetection:       true,
		},
		Variants: []Variant{
			{Name: "ours", Description: "default configuration (8KB buffer)", Adapters: oursDefaultAdapters()},
			{Name: "ours-4k", Description: "4KB streaming chunks", Adapters: oursAdaptersWithBuf(4096)},
		},
	},
	{
		Repo: "https://github.com/yuin/goldmark",
		Features: Features{
			Parser: "goldmark",
			Notes:  []string{"De facto standard Go Markdown parser", "Zero dependencies"},
		},
		Variants: []Variant{
			{Name: "goldmark", Description: "default configuration", Adapters: goldmarkAdapters()},
		},
	},
	{
		Repo: "https://github.com/charmbracelet/glamour",
		Features: Features{
			Parser:             "goldmark",
			TerminalRender:     true,
			SyntaxHighlighting: "Chroma",
			WordWrap:           "fixed width",
			Notes:              []string{"Uses goldmark internally", "Multiple built-in themes"},
		},
		Variants: []Variant{
			{Name: "glamour", Description: "default auto style", Adapters: glamourAdapters()},
		},
	},
	{
		Repo: "https://github.com/russross/blackfriday",
		Features: Features{
			Parser: "blackfriday",
			Notes:  []string{"Unmaintained since 2019", "Not CommonMark compliant"},
		},
		Variants: []Variant{
			{Name: "blackfriday", Description: "default configuration", Adapters: blackfridayAdapters()},
		},
	},
	{
		Repo: "https://github.com/gomarkdown/markdown",
		Features: Features{
			Parser: "gomarkdown",
			Notes:  []string{"Active fork of blackfriday"},
		},
		Variants: []Variant{
			{Name: "gomarkdown", Description: "CommonExtensions", Adapters: gomarkdownAdapters()},
		},
	},
	{
		Repo: "https://github.com/MichaelMure/go-term-markdown",
		Features: Features{
			Parser:             "blackfriday v1",
			TerminalRender:     true,
			SyntaxHighlighting: "Chroma v1",
			WordWrap:           "fixed width",
			Notes:              []string{"Inline terminal images via pixterm"},
		},
		Variants: []Variant{
			{Name: "go-term-md", Description: "80 columns", Adapters: goTermMdAdapters()},
		},
	},
}

// --- Adapter factories: ours ------------------------------------------------

func oursDefaultAdapters() Adapters {
	return Adapters{
		ParseFunc:      oursParse,
		RenderTerminal: oursRenderTerminal,
		RenderHTML:     oursRenderHTML,
	}
}

// oursAdaptersWithBuf returns adapters that use a specific buffer size
// for the streaming parser, giving all 3 capabilities (parse, terminal, HTML).
func oursAdaptersWithBuf(bufSize int) Adapters {
	return Adapters{
		ParseFunc: func(r io.Reader) (int, error) {
			events, err := markdown.Parse(r, markdown.WithBufSize(bufSize))
			if err != nil {
				return 0, err
			}
			return len(events), nil
		},
		RenderTerminal: func(r io.Reader, w io.Writer) error {
			sr := terminal.NewStreamRenderer(w, terminal.WithAnsi(terminal.AnsiOn))
			buf := make([]byte, bufSize)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					if _, werr := sr.Write(buf[:n]); werr != nil {
						return werr
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
			}
			return sr.Flush()
		},
		RenderHTML: func(r io.Reader, w io.Writer) error {
			events, err := markdown.Parse(r, markdown.WithBufSize(bufSize))
			if err != nil {
				return err
			}
			return html.Render(w, events, html.WithUnsafe())
		},
	}
}

func oursParse(r io.Reader) (int, error) {
	events, err := markdown.Parse(r)
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

func oursRenderTerminal(r io.Reader, w io.Writer) error {
	src, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return markdown.RenderToWriter(w, string(src), terminal.WithAnsi(terminal.AnsiOn))
}

func oursRenderHTML(r io.Reader, w io.Writer) error {
	events, err := markdown.Parse(r)
	if err != nil {
		return err
	}
	return html.Render(w, events, html.WithUnsafe())
}

// --- Adapter factories: goldmark --------------------------------------------

func goldmarkAdapters() Adapters {
	md := goldmark.New()
	mdHTML := goldmark.New(
		goldmark.WithExtensions(goldmarkext.GFM),
		goldmark.WithRendererOptions(
			goldmarkhtml.WithXHTML(),
			goldmarkhtml.WithUnsafe(),
		),
	)
	return Adapters{
		ParseFunc: func(r io.Reader) (int, error) {
			src, err := io.ReadAll(r)
			if err != nil {
				return 0, err
			}
			doc := md.Parser().Parse(goldmarktext.NewReader(src))
			// Count child nodes as a rough output metric.
			count := 0
			for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
				count++
			}
			return count, nil
		},
		RenderHTML: func(r io.Reader, w io.Writer) error {
			src, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			return mdHTML.Convert(src, w)
		},
	}
}

// --- Adapter factories: glamour ---------------------------------------------

func glamourAdapters() Adapters {
	var renderer *glamour.TermRenderer
	return Adapters{
		RenderTerminal: func(r io.Reader, w io.Writer) error {
			if renderer == nil {
				var err error
				renderer, err = glamour.NewTermRenderer(glamour.WithAutoStyle())
				if err != nil {
					return err
				}
			}
			src, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			out, err := renderer.Render(string(src))
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, out)
			return err
		},
	}
}

// --- Adapter factories: blackfriday -----------------------------------------

func blackfridayAdapters() Adapters {
	return Adapters{
		ParseFunc: func(r io.Reader) (int, error) {
			src, err := io.ReadAll(r)
			if err != nil {
				return 0, err
			}
			// blackfriday.Run parses + renders; no separate parse API.
			out := blackfriday.Run(src)
			return len(out), nil
		},
		RenderHTML: func(r io.Reader, w io.Writer) error {
			src, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			_, err = w.Write(blackfriday.Run(src))
			return err
		},
	}
}

// --- Adapter factories: gomarkdown ------------------------------------------

func gomarkdownAdapters() Adapters {
	return Adapters{
		ParseFunc: func(r io.Reader) (int, error) {
			src, err := io.ReadAll(r)
			if err != nil {
				return 0, err
			}
			p := gomarkdownparser.NewWithExtensions(gomarkdownparser.CommonExtensions)
			doc := p.Parse(src)
			renderer := gomarkdownhtml.NewRenderer(gomarkdownhtml.RendererOptions{})
			out := gomarkdown.Render(doc, renderer)
			return len(out), nil
		},
		RenderHTML: func(r io.Reader, w io.Writer) error {
			src, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			p := gomarkdownparser.NewWithExtensions(gomarkdownparser.CommonExtensions)
			doc := p.Parse(src)
			renderer := gomarkdownhtml.NewRenderer(gomarkdownhtml.RendererOptions{})
			_, err = w.Write(gomarkdown.Render(doc, renderer))
			return err
		},
	}
}

// --- Adapter factories: go-term-markdown ------------------------------------

func goTermMdAdapters() Adapters {
	return Adapters{
		RenderTerminal: func(r io.Reader, w io.Writer) error {
			src, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			out := goterm.Render(string(src), 80, 0)
			_, err = w.Write(out)
			return err
		},
	}
}
