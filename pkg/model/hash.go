package model

import (
	"crypto/sha1"
	"encoding/hex"
)

type Hash [sha1.Size]byte

func (h Hash) HexEncoding() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) String() string {
	return h.HexEncoding()
}
