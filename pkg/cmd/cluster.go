package cmd

import (
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/spf13/cobra"
)

func NewClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "cluster",
		Short:                 "Manage cluster configuration",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewClusterConfigCommand())

	return cmd
}

func NewClusterConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "config",
		Short:                 "Create cluster config",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			directory, _ := cmd.Flags().GetString("directory")

			planOpts := taskutil.DefaultPlanOptions{SpecURL: os.Getenv(taskutil.SpecURLEnvName)}
			task := task.NewTaskInterface(planOpts)
			return task.ProcessClusters(directory)
		},
	}

	cmd.Flags().StringP("directory", "d", "", "configuration output directory")

	return cmd
}
