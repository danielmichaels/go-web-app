package commands

import (
	"context"

	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) int {
	rootCmd := &cobra.Command{
		Use:     "familyhub",
		Aliases: []string{"fh"},
		Short:   "The family hub CLI",
	}

	rootCmd.AddCommand(ServeCmd(ctx))
	rootCmd.AddCommand(MigrateCmd(ctx))
	rootCmd.AddCommand(SendMailCmd(ctx))

	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}
