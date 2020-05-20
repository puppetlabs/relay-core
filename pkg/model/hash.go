package model

import (
	"crypto/sha1"

	"github.com/puppetlabs/nebula-tasks/pkg/util/hashutil"
)

// Hash wraps the result of a SHA-1 checksum operation. It is different than
// hashutil.Sum so that it can be used as a map key.
type Hash [sha1.Size]byte

func (h Hash) HexEncoding() string {
	return hashutil.Sum(h[:]).HexEncoding()
}

func (h Hash) String() string {
	return h.HexEncoding()
}
