package cmd

import (
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/spf13/cobra"
)

func NewFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "file",
		Short:                 "WriteFile specification data to a file",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			path, _ := cmd.Flags().GetString("path")
			output, _ := cmd.Flags().GetString("output")

			planOpts := taskutil.DefaultPlanOptions{SpecURL: os.Getenv(taskutil.SpecURLEnvName)}
			task := task.NewTaskInterface(planOpts)
			return task.WriteFile(file, path, output)
		},
	}

	cmd.Flags().StringP("file", "f", "", "file name")
	cmd.Flags().StringP("path", "p", "", "specification data path")
	cmd.Flags().StringP("output", "o", "", "output type")

	return cmd
}
