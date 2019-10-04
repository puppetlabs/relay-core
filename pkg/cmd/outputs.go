package cmd

import (
	"bytes"
	"context"
	"fmt"

	outputsclient "github.com/puppetlabs/nebula-tasks/pkg/outputs/client"
	"github.com/spf13/cobra"
)

func NewOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "output",
		Short:                 "Manage data that needs to be accessible to other tasks",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewSetOutputCommand())
	cmd.AddCommand(NewGetOutputCommand())

	return cmd
}

func NewSetOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "set",
		Short:                 "Set a value to a key that can be fetched by another task",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := outputsclient.NewDefaultOutputsClientFromNebulaEnv()
			if err != nil {
				return err
			}

			key, err := cmd.Flags().GetString("key")
			if err != nil {
				return err
			}

			value, err := cmd.Flags().GetString("value")
			if err != nil {
				return err
			}

			if err := client.SetOutput(context.Background(), key, value); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Successfully set output.")

			return nil
		},
	}

	cmd.Flags().StringP("key", "k", "", "the output key")
	cmd.Flags().StringP("value", "v", "", "the output value")

	return cmd
}

func NewGetOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "get",
		Short:                 "Get a value that a previous task stored",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := outputsclient.NewDefaultOutputsClientFromNebulaEnv()
			if err != nil {
				return err
			}

			taskName, err := cmd.Flags().GetString("task-name")
			if err != nil {
				return err
			}

			key, err := cmd.Flags().GetString("key")
			if err != nil {
				return err
			}

			value, err := client.GetOutput(context.Background(), taskName, key)
			if err != nil {
				return err
			}

			buf := bytes.NewBufferString(value)

			if _, err := buf.WriteTo(cmd.OutOrStdout()); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("task-name", "n", "", "the name of the task")
	cmd.Flags().StringP("key", "k", "", "the output key")

	return cmd
}
