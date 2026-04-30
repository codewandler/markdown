package bubbleview

import (
	"bytes"

	"github.com/codewandler/markdown/terminal"
)

type rendererState struct {
	cfg      config
	width    int
	source   bytes.Buffer
	output   bytes.Buffer
	renderer *terminal.StreamRenderer
	flushed  bool
}

func newRendererState(cfg config, width int) *rendererState {
	r := &rendererState{cfg: cfg, width: width}
	r.resetRenderer()
	return r
}

func (r *rendererState) append(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	_, _ = r.source.Write(p)
	if r.flushed {
		return r.rerender(r.width, false)
	}
	_, err := r.renderer.Write(p)
	return err
}

func (r *rendererState) flush() error {
	if r.flushed {
		return nil
	}
	if err := r.renderer.Flush(); err != nil {
		return err
	}
	r.flushed = true
	return nil
}

func (r *rendererState) reset(markdown []byte, flush bool) error {
	r.source.Reset()
	r.output.Reset()
	r.flushed = false
	r.resetRenderer()
	if len(markdown) > 0 {
		_, _ = r.source.Write(markdown)
		if _, err := r.renderer.Write(markdown); err != nil {
			return err
		}
	}
	if flush {
		return r.flush()
	}
	return nil
}

func (r *rendererState) rerender(width int, flush bool) error {
	if width < 0 {
		width = 0
	}
	r.width = width
	markdown := append([]byte(nil), r.source.Bytes()...)
	return r.reset(markdown, flush)
}

func (r *rendererState) string() string {
	return r.output.String()
}

func (r *rendererState) resetRenderer() {
	r.output.Reset()
	r.renderer = terminal.NewStreamRenderer(&r.output, r.cfg.rendererOptionsFor(r.width)...)
}
