package hashutil

import "encoding/hex"

type Sum []byte

func (s Sum) HexEncoding() string {
	return hex.EncodeToString(s)
}

func (s Sum) String() string {
	return s.HexEncoding()
}
