package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitworkspacefun/internal/gitcli"
	"gitworkspacefun/internal/repository"
	"gitworkspacefun/internal/workspace"

	"github.com/spf13/cobra"
)

var pullWorkers int

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Fast-forward clean repositories that are behind",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, repositories, err := loadRepositories()
		if err != nil {
			return err
		}
		results := workspace.ForEach(repositories, pullWorkers, func(repo repository.Repository) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			changes, err := gitcli.GetStatus(ctx, repo.Path)
			if err != nil {
				return err
			}
			if changes.IsDirty() {
				return fmt.Errorf("skipped: working tree contains changes")
			}
			syncStatus, err := gitcli.GetSyncStatus(ctx, repo.Path)
			if err != nil {
				return err
			}
			switch syncStatus.State {
			case gitcli.SyncNoRemote:
				return fmt.Errorf("skipped: no upstream branch")
			case gitcli.SyncDiverged:
				return fmt.Errorf("skipped: branch has diverged")
			case gitcli.SyncClean, gitcli.SyncAhead:
				return fmt.Errorf("skipped: already up to date")
			case gitcli.SyncBehind:
				if err := gitcli.PullFastForward(ctx, repo.Path); err != nil {
					return err
				}
				return nil
			default:
				return fmt.Errorf("skipped: unknown sync state %q", syncStatus.State)
			}
		})
		failed := 0
		for _, result := range results {
			if result.Error != nil {
				if !strings.HasPrefix(result.Error.Error(), "skipped:") {
					failed++
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n⚠ %v\n\n", result.Repository.Name, result.Error)
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n✓ updated\n\n", result.Repository.Name)
		}
		if failed > 0 {
			return fmt.Errorf("%d repository pulls were skipped or failed", failed)
		}
		return nil
	},
}

func init() {
	pullCmd.Flags().IntVar(&pullWorkers, "workers", 4, "maximum concurrent pulls")
	rootCmd.AddCommand(pullCmd)
}
