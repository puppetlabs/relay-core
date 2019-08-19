package cmd

import (
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/spf13/cobra"
)

func NewCredentialsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "credentials",
		Short:                 "Manage credentials configuration",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewCredentialsConfigCommand())

	return cmd
}

func NewCredentialsConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "config",
		Short:                 "Create credentials configuration",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			directory, _ := cmd.Flags().GetString("directory")

			planOpts := taskutil.DefaultPlanOptions{SpecURL: os.Getenv(taskutil.SpecURLEnvName)}
			task := task.NewTaskInterface(planOpts)
			return task.ProcessCredentials(directory)
		},
	}

	cmd.Flags().StringP("directory", "d", "", "configuration output directory")

	return cmd
}
