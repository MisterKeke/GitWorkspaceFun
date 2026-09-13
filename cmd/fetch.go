package cmd

import (
	"context"
	"fmt"
	"time"

	"gitworkspacefun/internal/gitcli"
	"gitworkspacefun/internal/repository"
	"gitworkspacefun/internal/workspace"

	"github.com/spf13/cobra"
)

var fetchWorkers int

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch updates for all registered repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, repositories, err := loadRepositories()
		if err != nil {
			return err
		}
		results := workspace.ForEach(repositories, fetchWorkers, func(repo repository.Repository) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			return gitcli.Fetch(ctx, repo.Path)
		})
		failed := 0
		for _, result := range results {
			if result.Error != nil {
				failed++
				fmt.Fprintf(cmd.ErrOrStderr(), "✗ %-20s %v\n", result.Repository.Name, result.Error)
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", result.Repository.Name)
		}
		if failed > 0 {
			return fmt.Errorf("%d repository fetches failed", failed)
		}
		return nil
	},
}

func init() {
	fetchCmd.Flags().IntVar(&fetchWorkers, "workers", 4, "maximum concurrent fetches")
	rootCmd.AddCommand(fetchCmd)
}
