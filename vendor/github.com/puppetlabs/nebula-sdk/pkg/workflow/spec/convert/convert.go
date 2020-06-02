package convert

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/convert/render/jira"
)

type ConvertType string

func (ct ConvertType) String() string {
	return string(ct)
}

const (
	ConvertTypeHtml  ConvertType = "html"
	ConvertTypeJira  ConvertType = "jira"
	ConvertTypeSlack ConvertType = "slack"
)

func ConvertMarkdown(ct ConvertType, md []byte) ([]byte, error) {
	doc := markdown.Parse(md, nil)

	renderer, err := NewMarkdownRenderer(ct)
	if err != nil {
		return nil, err
	}

	return markdown.Render(doc, renderer), nil
}

func NewMarkdownRenderer(ct ConvertType) (markdown.Renderer, error) {
	switch ct {
	case ConvertTypeHtml:
		opts := html.RendererOptions{
			Flags: html.CommonFlags,
		}
		return html.NewRenderer(opts), nil
	case ConvertTypeJira:
		opts := jira.RendererOptions{}
		return jira.NewRenderer(opts), nil
	default:
		return nil, ErrConvertTypeNotSupported
	}
}
