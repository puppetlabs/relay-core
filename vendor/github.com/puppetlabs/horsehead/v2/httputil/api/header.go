package api

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	MaxContentDispositionHeaderFilenameLength = 256
	DefaultAttachmentFilename                 = "attachment"
)

var (
	contentDispositionASCIIPrintableFilter = regexp.MustCompile(`[^\w\.-]+`)
)

// SetContentDispositionHeader cleans the given filename, setting the sanitized
// name in the response as the Content-Disposition header filename value.
//
// Content-Disposition is so hard to get right that there's an entire RFC
// dedicated to it [https://tools.ietf.org/html/rfc6266]. It is recommended to
// just use this function and accept the resulting filename rather than trying
// to customize anything.
func SetContentDispositionHeader(w http.ResponseWriter, filename string) {
	// Fill in filename with default if empty.
	ext := path.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	if len(base) == 0 {
		filename = DefaultAttachmentFilename + ext
	}

	// transform.Chain transformers are not thread safe. Do not move this to a
	// global.
	diacriticRemover := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

	// Remove diacritics before sanitizing to ASCII.
	if tr, _, err := transform.String(diacriticRemover, filename); err != nil {
		// Bad filename. We'll try to save the extension and just return a
		// generic filename.
		ext := path.Ext(filename)
		ext, _, _ = transform.String(diacriticRemover, ext)

		filename = DefaultAttachmentFilename + ext
	} else {
		filename = tr
	}

	// Strip out all non-printable or non-ASCII characters.
	filename = contentDispositionASCIIPrintableFilter.ReplaceAllString(filename, "_")

	// Reduce filename to reasonable size, making sure not to clobber the
	// extension.
	ext = path.Ext(filename)
	base = filename[:len(filename)-len(ext)]

	maxLength := MaxContentDispositionHeaderFilenameLength - len(ext)

	// It is possible that the extension is in fact longer than the maximum
	// filename length, in which case we truncate the extension.
	if maxLength <= 0 {
		maxLength = 1
		ext = ext[:MaxContentDispositionHeaderFilenameLength-1]
	}

	// Once we know the final extension length, we truncate the rest of the
	// filename.
	if len(base) > maxLength {
		base = base[:maxLength]
	}

	filename = base + ext

	w.Header().Set("content-disposition", fmt.Sprintf(`attachment; filename=%s`, filename))
}
