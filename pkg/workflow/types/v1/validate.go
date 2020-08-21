package v1

import "github.com/puppetlabs/relay-core/pkg/util/typeutil"

// ValidateYAML validates a yaml document according to the schema specification
func ValidateYAML(y string) error {
	return typeutil.ValidateYAMLString(WorkflowSchema, y)
}
