package stringutil

// Uniques takes two sorted lists and removes duplicates. It does not modify the
// underlying list.
//
// Compare to POSIX `uniq` command.
func Uniques(items []string) []string {
	if len(items) < 2 {
		return items
	}

	var keep []string
	keep = append(keep, items[0])

	for i := 1; i < len(items); i++ {
		if items[i-1] == items[i] {
			continue
		}

		keep = append(keep, items[i])
	}

	return keep
}
