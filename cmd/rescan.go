package cmd

import (
	"fmt"
	"os"

	"gitworkspacefun/internal/config"
	"gitworkspacefun/internal/repository"

	"github.com/spf13/cobra"
)

var rescanCmd = &cobra.Command{
	Use:   "rescan",
	Short: "Rescan all configured workspaces",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if len(cfg.Workspaces) == 0 {
			return fmt.Errorf("no workspaces configured; run gw scan <directory> first")
		}
		found := make([]repository.Repository, 0)
		for _, root := range cfg.Workspaces {
			if _, statErr := os.Stat(root); os.IsNotExist(statErr) {
				continue
			}
			repositories, scanErr := repository.Scan(root)
			if scanErr != nil {
				return fmt.Errorf("rescan %s: %w", root, scanErr)
			}
			for _, repo := range repositories {
				if !repository.ContainsRepository(found, repo.Path) {
					found = append(found, repo)
				}
			}
		}
		for _, existing := range cfg.Repositories {
			if _, statErr := os.Stat(existing.Path); os.IsNotExist(statErr) && !repository.ContainsRepository(found, existing.Path) {
				found = append(found, existing)
			}
		}
		cfg.Repositories = found
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Rescanned %d repositories across %d workspaces.\n", len(found), len(cfg.Workspaces))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rescanCmd)
}
