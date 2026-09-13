package cmd

import (
	"context"
	"fmt"
	"time"

	"gitworkspacefun/internal/gitcli"

	"github.com/spf13/cobra"
)

var forgottenDirty bool

var forgottenCmd = &cobra.Command{
	Use:   "forgotten",
	Short: "Show repositories with old commits",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, repositories, err := loadRepositories()
		if err != nil {
			return err
		}
		for _, repo := range repositories {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			commit, commitErr := gitcli.GetLatestCommit(ctx, repo.Path)
			changes, statusErr := gitcli.GetStatus(ctx, repo.Path)
			cancel()
			if commitErr != nil {
				return fmt.Errorf("get latest commit for %s: %w", repo.Name, commitErr)
			}
			if statusErr != nil {
				return fmt.Errorf("get status for %s: %w", repo.Name, statusErr)
			}
			if forgottenDirty && !changes.IsDirty() {
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", repo.Name, commit.Timestamp.Format("2006-01-02"), worktreeState(changes.IsDirty()))
		}
		return nil
	},
}

func init() {
	forgottenCmd.Flags().BoolVar(&forgottenDirty, "dirty", false, "show only dirty repositories")
	rootCmd.AddCommand(forgottenCmd)
}
