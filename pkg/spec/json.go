package spec

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/puppetlabs/leg/encoding/transfer"
)

func ParseJSON(r io.Reader) (any, error) {
	var tree any
	if err := json.NewDecoder(r).Decode(&tree); err != nil {
		return nil, err
	}

	return tree, nil
}

func ParseJSONString(data string) (any, error) {
	return ParseJSON(strings.NewReader(data))
}

type JSONTree struct {
	Tree any
}

func (jt *JSONTree) UnmarshalJSON(data []byte) error {
	tree, err := ParseJSONString(string(data))
	if err != nil {
		return err
	}

	*jt = JSONTree{Tree: tree}
	return nil
}

func (jt JSONTree) MarshalJSON() ([]byte, error) {
	return json.Marshal(transfer.JSONInterface{Data: jt.Tree})
}
