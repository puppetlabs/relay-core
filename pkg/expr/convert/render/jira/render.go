package jira

import (
	"io"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

type RendererOptions struct {
}

type Renderer struct {
}

func NewRenderer(opts RendererOptions) *Renderer {
	return nil
}

func (r *Renderer) RenderHeader(w io.Writer, ast ast.Node) {}

func (r *Renderer) RenderFooter(w io.Writer, ast ast.Node) {}

func (r *Renderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	switch node := node.(type) {
	case *ast.Code:
		r.code(w, node)
	case *ast.Image:
		r.image(w, node, entering)
	case *ast.Link:
		r.link(w, node, entering)
	case *ast.Text:
		r.text(w, node)
	}
	return ast.GoToNext
}

func (r *Renderer) code(w io.Writer, code *ast.Code) {
	io.WriteString(w, "{code}")
	w.Write(code.Literal)
	io.WriteString(w, "{code}")
}

func (r *Renderer) text(w io.Writer, text *ast.Text) {
	switch text.Parent.(type) {
	case *ast.Link:
		if string(text.Literal) != "" {
			io.WriteString(w, "[")
			w.Write(text.Literal)
		}
	case *ast.Image:
		io.WriteString(w, "\n\n")
		break
	default:
		// TODO These are specific workarounds that will need better handling
		r := regexp.MustCompile("(^|\\s+)(-\\s*-\\s*-)(\\s+|$)")
		md := r.ReplaceAllString(string(text.Literal), "\n----\n")
		md = strings.TrimPrefix(md, "%%% ")
		md = strings.TrimSuffix(md, " %%%")

		w.Write([]byte(md))
	}
}

func (r *Renderer) link(w io.Writer, link *ast.Link, entering bool) {
	if entering {
		r.linkEnter(w, link)
	} else {
		r.linkExit(w, link)
	}
}

func (r *Renderer) linkEnter(w io.Writer, link *ast.Link) {
}

func (r *Renderer) linkExit(w io.Writer, link *ast.Link) {
	io.WriteString(w, "|")
	w.Write(link.Destination)
	io.WriteString(w, "]")
}

func (r *Renderer) image(w io.Writer, image *ast.Image, entering bool) {
	if entering {
		r.imageEnter(w, image)
	} else {
		r.imageExit(w, image)
	}
}

func (r *Renderer) imageEnter(w io.Writer, image *ast.Image) {
}

func (r *Renderer) imageExit(w io.Writer, image *ast.Image) {
	switch image.Parent.(type) {
	case *ast.Link:
		io.WriteString(w, "[")
	}

	io.WriteString(w, "!")
	w.Write(image.Destination)
	io.WriteString(w, "!")
}
