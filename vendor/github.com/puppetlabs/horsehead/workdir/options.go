package workdir

import "os"

// Options for changing the behavior of directory management
type Options struct {
	// Mode is the octal filemode to use when creating each directory
	Mode os.FileMode
}
