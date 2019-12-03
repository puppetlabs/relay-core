package state

import "github.com/puppetlabs/horsehead/v2/encoding/transfer"

type StateEnvelope struct {
	Key   string             `json:"key"`
	Value transfer.JSONOrStr `json:"value"`
}
