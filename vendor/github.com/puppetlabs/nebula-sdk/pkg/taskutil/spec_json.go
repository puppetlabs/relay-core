package taskutil

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
)

// JSONSpec implements a custom unmarshaler of any JSON value type that handles
// automatically decoding `{"$encoding": "base64", "data": ...}` from the spec.
type JSONSpec struct {
	Value interface{}
}

func (js *JSONSpec) UnmarshalJSON(data []byte) error {
	switch data[0] {
	case '{':
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}

		// Check for encoded binary.
		if e, d := m["$encoding"], m["data"]; len(e) > 0 && len(d) > 0 && e[0] == '"' && d[0] == '"' {
			var ev transfer.JSON
			if err := json.Unmarshal(data, &ev); err != nil {
				return err
			}

			v, err := ev.Decode()
			if err != nil {
				return err
			}

			js.Value = string(v)
		} else {
			// Otherwise we just want to unmarshal this as usual, with the caveat
			// that each value should also be handled as a JSONSpec.
			vm := make(map[string]interface{}, len(m))
			for k, data := range m {
				var v JSONSpec
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}

				vm[k] = v.Value
			}

			js.Value = vm
		}

		return nil
	case '[':
		var l []JSONSpec
		if err := json.Unmarshal(data, &l); err != nil {
			return err
		}

		vl := make([]interface{}, len(l))
		for i, v := range l {
			vl[i] = v.Value
		}

		js.Value = vl

		return nil
	default:
		return json.Unmarshal(data, &js.Value)
	}
}

type JSONSpecDecoder struct{}

var _ SpecDecoder = &JSONSpecDecoder{}

func (u JSONSpecDecoder) DecodeSpec(r io.Reader) (interface{}, error) {
	var root JSONSpec
	if err := json.NewDecoder(r).Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %+v", err)
	}

	return root.Value, nil
}
