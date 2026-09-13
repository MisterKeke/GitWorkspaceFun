package cmd

import (
	"fmt"
	"os"

	"gitworkspacefun/internal/config"

	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove missing repositories from the configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		kept := cfg.Repositories[:0]
		removed := 0
		for _, repo := range cfg.Repositories {
			if _, statErr := os.Stat(repo.Path); os.IsNotExist(statErr) {
				removed++
				continue
			}
			kept = append(kept, repo)
		}
		cfg.Repositories = kept
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed %d missing repositories.\n", removed)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
