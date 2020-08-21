package input

import (
	"context"
	"io"
)

// FileManager is a provider for workflow step InputFiles
type FileManager interface {
	// GetByURL takes a url string, validates it as a url and returns the content
	// from that URL as an io.Reader. Any status code other than 200 will cause
	// it to return an error and it does NOT follow redirects.
	// InputFiles are fetched during the creation of a run, so changes these
	// remote files can effect the predictability of subsequent runs of the
	// same workflow revision.
	GetByURL(ctx context.Context, url string) (io.Reader, error)
}
