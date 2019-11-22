package fnlib

import "github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/fn"

var (
	library = map[string]fn.Descriptor{
		"append":        appendDescriptor,
		"concat":        concatDescriptor,
		"jsonUnmarshal": jsonUnmarshalDescriptor,
		"merge":         mergeDescriptor,
	}
)

func Library() fn.Map {
	return fn.NewMap(library)
}
