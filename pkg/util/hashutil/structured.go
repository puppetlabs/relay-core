package hashutil

import (
	"encoding/json"
	"hash"
)

type StructuredHash struct {
	factory func() hash.Hash
	data    map[string][]string
}

func (sh *StructuredHash) Set(name string, values ...string) {
	sh.data[name] = values
}

func (sh *StructuredHash) Sum() (Sum, error) {
	b, err := json.Marshal(sh.data)
	if err != nil {
		return nil, err
	}

	h := sh.factory()
	h.Write(b)
	return Sum(h.Sum(nil)), nil
}

func NewStructuredHash(factory func() hash.Hash) *StructuredHash {
	return &StructuredHash{
		factory: factory,
		data:    make(map[string][]string),
	}
}
