package cmd

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() (*cobra.Command, error) {
	c := &cobra.Command{
		Use:           "ni",
		Short:         "Nebula Interface",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return c, nil
}
