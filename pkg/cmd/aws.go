package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/spf13/cobra"
)

func NewAWSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "aws",
		Short:                 "Manage AWS access",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewAWSConfigCommand())
	cmd.AddCommand(NewAWSEnvCommand())

	return cmd
}

func NewAWSConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "config",
		Short:                 "Create an AWS configuration suitable for using with an AWS CLI or SDK",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			directory, _ := cmd.Flags().GetString("directory")

			planOpts := taskutil.DefaultPlanOptions{SpecURL: os.Getenv(taskutil.SpecURLEnvName)}
			task := task.NewTaskInterface(planOpts)
			return task.ProcessAWS(directory)
		},
	}

	cmd.Flags().StringP("directory", "d", "", "configuration output directory")

	return cmd
}

func NewAWSEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "env",
		Short:                 "Create a POSIX-compatible script that can be sourced to configure the AWS CLI",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			directory, _ := cmd.Flags().GetString("directory")
			if directory == "" {
				directory = filepath.Join(task.DefaultPath, ".aws")
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				`export AWS_CONFIG_FILE=%s
export AWS_SHARED_CREDENTIALS_FILE=%s
`,
				quoteShell(filepath.Join(directory, "config")),
				quoteShell(filepath.Join(directory, "credentials")),
			)
			return nil
		},
	}

	cmd.Flags().StringP("directory", "d", "", "configuration output directory")

	return cmd
}

func quoteShell(data string) string {
	return `'` + strings.Replace(data, `'`, `'"'"'`, -1) + `'`
}
