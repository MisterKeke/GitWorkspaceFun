package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitworkspacefun/internal/config"
	"gitworkspacefun/internal/gitcli"
	"gitworkspacefun/internal/repository"

	"github.com/spf13/cobra"
)

var dirtyCmd = &cobra.Command{
	Use:   "dirty [repository]",
	Short: "Show uncommitted changes",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		repositories := cfg.Repositories
		if len(args) == 1 {
			repo, err := repository.FindByName(cfg.Repositories, args[0])
			if err != nil {
				return err
			}
			repositories = []repository.Repository{*repo}
		}

		out := cmd.OutOrStdout()
		dirtyRepositories := 0
		for _, repo := range repositories {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			changes, statusErr := gitcli.GetStatus(ctx, repo.Path)
			cancel()
			if statusErr != nil {
				return fmt.Errorf("get status for %s: %w", repo.Name, statusErr)
			}
			if !changes.IsDirty() {
				continue
			}

			dirtyRepositories++
			fmt.Fprintln(out, repo.Name)
			for _, line := range changes.Lines {
				fmt.Fprintf(out, "  %s\n", strings.TrimLeft(line, " "))
			}
			fmt.Fprintln(out)
		}

		if dirtyRepositories == 0 {
			if len(args) == 1 {
				fmt.Fprintf(out, "%s is clean.\n", repositories[0].Name)
			} else {
				fmt.Fprintln(out, "No dirty repositories.")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dirtyCmd)
}
