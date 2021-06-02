package serialize

import (
	"encoding/json"

	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
)

type JSONTree struct {
	parse.Tree
}

func (jt *JSONTree) UnmarshalJSON(data []byte) error {
	tree, err := parse.ParseJSONString(string(data))
	if err != nil {
		return err
	}

	*jt = JSONTree{Tree: tree}
	return nil
}

func (jt JSONTree) MarshalJSON() ([]byte, error) {
	return json.Marshal(transfer.JSONInterface{Data: jt.Tree})
}
