package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/MisterKeke/GitWorkspaceFun/internal/gitcli"

	"github.com/spf13/cobra"
)

var staleDays int

var staleCmd = &cobra.Command{
	Use:   "stale",
	Short: "Show repositories whose latest commit is old",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, repositories, err := loadRepositories()
		if err != nil {
			return err
		}
		cutoff := time.Now().AddDate(0, 0, -staleDays)
		for _, repo := range repositories {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			commit, commitErr := gitcli.GetLatestCommit(ctx, repo.Path)
			cancel()
			if commitErr != nil {
				return fmt.Errorf("get latest commit for %s: %w", repo.Name, commitErr)
			}
			if commit.Timestamp.Before(cutoff) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", repo.Name, commit.Timestamp.Format("2006-01-02"))
			}
		}
		return nil
	},
}

func init() {
	staleCmd.Flags().IntVar(&staleDays, "days", 30, "minimum age in days")
	rootCmd.AddCommand(staleCmd)
}
