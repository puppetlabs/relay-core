package stringutil

import (
	"sort"
)

// Diff examines two unordered lists of strings and calculates the items that
// should be removed (exist only in the first argument) and added (exist only in
// the second argument).
//
// Compare to POSIX `comm` command.
func Diff(prev, next []string) (added []string, removed []string) {
	// Copy input so sorting doesn't mess with it.
	prev = append([]string(nil), prev...)
	next = append([]string(nil), next...)

	// Sort.
	sort.Strings(prev)
	sort.Strings(next)

	// Deduplicate.
	prev = Uniques(prev)
	next = Uniques(next)

	pi, ni := 0, 0
	pl, nl := len(prev), len(next)

	for pi < pl && ni < nl {
		if prev[pi] < next[ni] {
			removed = append(removed, prev[pi])
			pi++
		} else if prev[pi] > next[ni] {
			added = append(added, next[ni])
			ni++
		} else {
			pi++
			ni++
		}
	}

	removed = append(removed, prev[pi:]...)
	added = append(added, next[ni:]...)

	return
}
