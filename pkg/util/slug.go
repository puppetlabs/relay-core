package util

import (
	"math"
	"regexp"
	"strings"
)

var (
	slugexp   = regexp.MustCompile(`[^\w\s]+`)
	spacesexp = regexp.MustCompile(`\s+`)
)

func Slug(name string) string {
	slg := spacesexp.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	slg = slugexp.ReplaceAllString(slg, "-")

	len := float64(len(slg))
	maxLen := math.Min(len, 63)

	return strings.Trim(slg[0:int(maxLen)], "-")
}
