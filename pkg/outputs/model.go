package outputs

import "github.com/puppetlabs/horsehead/v2/encoding/transfer"

type Output struct {
	TaskName string
	Key      string
	Value    transfer.JSONInterface
}
