// Package html renders stream.Event slices as CommonMark-compliant HTML.
//
// The renderer walks events linearly and does not re-parse Markdown
// syntax. It operates on a complete event slice (non-streaming).
//
// Basic usage:
//
//	events, _ := markdown.Parse(r)
//	out, _ := html.RenderString(events)
//
// For raw HTML passthrough (required for full CommonMark compliance):
//
//	out, _ := html.RenderString(events, html.WithUnsafe())
//
// The default output style is XHTML (self-closing void elements like
// <br /> and <hr />). Use [WithHTML5] for HTML5 style (<br>, <hr>).
package html
