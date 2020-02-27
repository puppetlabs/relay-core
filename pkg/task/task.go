package task

import "crypto/sha1"

// Metadata represents task metadata (such as the hash uniquely identifying the
// task).
type Metadata struct {
	Name string
	Hash [sha1.Size]byte
}
