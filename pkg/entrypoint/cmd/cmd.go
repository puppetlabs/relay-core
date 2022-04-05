package cmd

import (
	"github.com/puppetlabs/relay-core/pkg/model"
)

type Command interface {
	Execute(args []string) error
}

func NewMap() map[string]Command {
	return map[string]Command{
		model.ToolsCommandInitialize: NewInitializeCommand(),
	}
}
