package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"
	"github.com/MisterKeke/GitWorkspaceFun/internal/gitcli"

	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Show the current branch for each repository",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "REPOSITORY\tBRANCH")
		for _, repo := range cfg.Repositories {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			branch, branchErr := gitcli.GetBranch(ctx, repo.Path)
			cancel()
			if branchErr != nil {
				return fmt.Errorf("get branch for %s: %w", repo.Name, branchErr)
			}

			fmt.Fprintf(writer, "%s\t%s\n", repo.Name, branch)
		}

		return writer.Flush()
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)
}
