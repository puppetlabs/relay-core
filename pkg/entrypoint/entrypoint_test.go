package entrypoint_test

import (
	"testing"

	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
)

// TODO Consider replacing with proper unit testing instead of integration testing
// Not all of these examples would be functional if used in a live environment (i.e. `hashicorp/http-echo` does not have `echo`)
func TestImageEntrypoint(t *testing.T) {
	tcs := []struct {
		Name         string
		Image        string
		Command      []string
		Args         []string
		ExpectedArgs []string
	}{
		{
			Name:         "entrypoint:false;command:true;command_override:false;args:false",
			Image:        "busybox",
			Command:      nil,
			Args:         nil,
			ExpectedArgs: []string{model.EntrypointCommandFlag, "sh", model.EntrypointCommandArgSeparator},
		},
		{
			Name:         "entrypoint:false;command:true;command_override:true;args:false",
			Image:        "busybox",
			Command:      []string{"echo", "hello"},
			Args:         nil,
			ExpectedArgs: []string{model.EntrypointCommandFlag, "echo", model.EntrypointCommandArgSeparator, "hello"},
		},
		{
			Name:         "entrypoint:false;command:true;command_override:false;args:true",
			Image:        "busybox",
			Command:      nil,
			Args:         []string{"echo", "hello"},
			ExpectedArgs: []string{model.EntrypointCommandFlag, "sh", model.EntrypointCommandArgSeparator, "echo", "hello"},
		},
		{
			Name:         "entrypoint:false;command:true;command_override:true;args:true",
			Image:        "busybox",
			Command:      []string{"echo"},
			Args:         []string{"hello"},
			ExpectedArgs: []string{model.EntrypointCommandFlag, "echo", model.EntrypointCommandArgSeparator, "hello"},
		},
		{
			Name:         "entrypoint:true;command:true;command_override:false;args:false",
			Image:        "nginx",
			Command:      nil,
			Args:         nil,
			ExpectedArgs: []string{model.EntrypointCommandFlag, "/docker-entrypoint.sh", model.EntrypointCommandArgSeparator, "nginx", "-g", "daemon off;"},
		},
		{
			Name:         "entrypoint:true;command:true;command_override:true;args:false",
			Image:        "nginx",
			Command:      []string{"echo", "hello"},
			Args:         nil,
			ExpectedArgs: []string{model.EntrypointCommandFlag, "echo", model.EntrypointCommandArgSeparator, "hello"},
		},
		{
			Name:         "entrypoint:true;command:true;command_override:false;args:true",
			Image:        "nginx",
			Command:      nil,
			Args:         []string{"echo", "hello"},
			ExpectedArgs: []string{model.EntrypointCommandFlag, "/docker-entrypoint.sh", model.EntrypointCommandArgSeparator, "echo", "hello"},
		},
		{
			Name:         "entrypoint:true;command:true;command_override:true;args:true",
			Image:        "nginx",
			Command:      []string{"echo"},
			Args:         []string{"hello"},
			ExpectedArgs: []string{model.EntrypointCommandFlag, "echo", model.EntrypointCommandArgSeparator, "hello"},
		},
		{
			Name:         "entrypoint:true;command:false;command_override:false;args:false",
			Image:        "hashicorp/http-echo",
			Command:      nil,
			Args:         nil,
			ExpectedArgs: []string{model.EntrypointCommandFlag, "/http-echo", model.EntrypointCommandArgSeparator},
		},
		{
			Name:         "entrypoint:true;command:false;command_override:true;args:false",
			Image:        "hashicorp/http-echo",
			Command:      []string{"echo", "hello"},
			Args:         nil,
			ExpectedArgs: []string{model.EntrypointCommandFlag, "echo", model.EntrypointCommandArgSeparator, "hello"},
		},
		{
			Name:         "entrypoint:true;command:false;command_override:false;args:true",
			Image:        "hashicorp/http-echo",
			Command:      nil,
			Args:         []string{"-text", "hello"},
			ExpectedArgs: []string{model.EntrypointCommandFlag, "/http-echo", model.EntrypointCommandArgSeparator, "-text", "hello"},
		},
		{
			Name:         "entrypoint:true;command:false;command_override:true;args:true",
			Image:        "hashicorp/http-echo",
			Command:      []string{"echo"},
			Args:         []string{"hello"},
			ExpectedArgs: []string{model.EntrypointCommandFlag, "echo", model.EntrypointCommandArgSeparator, "hello"},
		},
	}

	for _, test := range tcs {
		t.Run(test.Name, func(t *testing.T) {
			ep, err := entrypoint.ImageEntrypoint(test.Image, test.Command, test.Args)
			require.NoError(t, err)

			require.Equal(t, test.ExpectedArgs, ep.Args)
		})
	}
}
