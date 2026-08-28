package web

// This file holds the one place Markdown becomes HTML. Issue bodies are
// Markdown by contract (ADR 0001), so the web renders them as prose rather
// than as source. goldmark earns its place as the dependency here because a
// hand-rolled parser is exactly the kind of correctness and injection surface
// this project refuses to maintain. Rendering is safe by default: goldmark
// escapes raw HTML and dangerous link schemes out of the output, so a file
// written by anything, a hand, an agent, or a merge, cannot script the page.

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// markdown is the shared converter: CommonMark plus the GitHub extensions
// (tables, strikethrough, task lists, autolinks) an issue tracker's prose
// actually uses.
var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

// renderMarkdown turns one Markdown source into the HTML the page embeds. A
// source the converter cannot handle falls back to the escaped text in a
// paragraph: degraded, never dropped (ADR 0003).
func renderMarkdown(src string) template.HTML {
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(src), &buf); err != nil {
		return template.HTML("<p>" + template.HTMLEscapeString(src) + "</p>") //nolint:gosec // escaped above
	}
	return template.HTML(buf.String()) //nolint:gosec // goldmark escapes raw HTML by default
}
