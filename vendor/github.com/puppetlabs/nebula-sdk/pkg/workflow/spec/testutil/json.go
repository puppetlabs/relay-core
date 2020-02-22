package testutil

import (
	"fmt"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
)

func JSONSecret(name string) map[string]interface{} {
	return map[string]interface{}{"$type": "Secret", "name": name}
}

func JSONOutput(from, name string) map[string]interface{} {
	return map[string]interface{}{"$type": "Output", "from": from, "name": name}
}

func JSONParameter(name string) map[string]interface{} {
	return map[string]interface{}{"$type": "Parameter", "name": name}
}

func JSONInvocation(name string, args interface{}) map[string]interface{} {
	return map[string]interface{}{fmt.Sprintf("$fn.%s", name): args}
}

func JSONEncoding(ty transfer.EncodingType, data interface{}) map[string]interface{} {
	return map[string]interface{}{"$encoding": string(ty), "data": data}
}
