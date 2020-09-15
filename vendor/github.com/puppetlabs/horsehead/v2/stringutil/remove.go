package stringutil

import (
	"strings"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
)

func RemoveAll(s string, cutset string) string {
	return RemoveAllFunc(s, func(r rune) bool {
		return strings.IndexRune(cutset, r) >= 0
	})
}

func RemoveAllFunc(s string, f func(rune) bool) string {
	r, _, _ := transform.String(runes.Remove(runes.Predicate(f)), s)
	return r
}
