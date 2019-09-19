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

	c.AddCommand(NewClusterCommand())
	c.AddCommand(NewCredentialsCommand())
	c.AddCommand(NewFileCommand())
	c.AddCommand(NewGetCommand())
	c.AddCommand(NewGitCommand())
	c.AddCommand(NewAWSCommand())
	c.AddCommand(NewOutputCommand())

	return c, nil
}
