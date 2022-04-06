package cmd

import (
	"errors"
	"os"
	"path"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/file"
)

type copyTarget struct {
	destination string
	source      string
}

type InitializeCommand struct {
	entrypoint copyTarget
}

func (ic *InitializeCommand) Execute(args []string) error {
	if _, err := os.Stat(ic.entrypoint.destination); errors.Is(err, os.ErrNotExist) {
		if err := file.Copy(ic.entrypoint.source, ic.entrypoint.destination, 0111); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func NewInitializeCommand() *InitializeCommand {
	return &InitializeCommand{
		entrypoint: copyTarget{
			destination: path.Join(model.ToolsMountPath, model.EntrypointCommand),
			source:      model.ToolsSource,
		},
	}
}
