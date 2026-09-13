package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"gitworkspacefun/internal/gitcli"

	"github.com/spf13/cobra"
)

var logLimit int
var logJSON bool

var logCmd = &cobra.Command{
	Use:   "log <repository>",
	Short: "Show recent commits",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := loadRepositories()
		if err != nil {
			return err
		}
		repositories, err := selectRepositories(cfg, args[0])
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		commits, err := gitcli.GetLog(ctx, repositories[0].Path, logLimit)
		if err != nil {
			return fmt.Errorf("get log for %s: %w", repositories[0].Name, err)
		}
		if logJSON {
			return writeJSON(cmd.OutOrStdout(), commits)
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		for _, commit := range commits {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", commit.Hash[:minInt(7, len(commit.Hash))], commit.Message, commit.Author)
		}
		return writer.Flush()
	},
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	logCmd.Flags().IntVar(&logLimit, "limit", 5, "number of commits to show")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "print JSON output")
	rootCmd.AddCommand(logCmd)
}
