package v1

import (
	"github.com/puppetlabs/relay-core/pkg/util/typeutil"
	"github.com/puppetlabs/relay-core/pkg/workflow/asset"
	"github.com/xeipuuv/gojsonschema"
)

var WorkflowSchema *gojsonschema.Schema

func init() {
	workflowSchema, err := typeutil.LoadSchemaFromStrings(asset.MustAssetString("schemas/v1/Workflow.json"))

	if err != nil {
		panic(err)
	}

	WorkflowSchema = workflowSchema
}
