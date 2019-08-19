package cmd

import (
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/spf13/cobra"
)

func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "get",
		Short:                 "Get specification data",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("path")

			planOpts := taskutil.DefaultPlanOptions{SpecURL: os.Getenv(taskutil.SpecURLEnvName)}
			task := task.NewTaskInterface(planOpts)
			data, err := task.ReadData(path)
			if err != nil {
				return err
			}

			cmd.OutOrStdout().Write(data)

			return nil
		},
	}

	cmd.Flags().StringP("path", "p", "", "specification data path")

	return cmd
}
