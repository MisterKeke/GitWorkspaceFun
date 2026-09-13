package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"
	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scan a directory for Git repositories",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		directory, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolve scan directory: %w", err)
		}
		repositories, err := repository.Scan(directory)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		workspaceAdded := false
		if !containsString(cfg.Workspaces, directory) {
			cfg.Workspaces = append(cfg.Workspaces, directory)
			workspaceAdded = true
		}

		newCount := 0
		knownCount := 0
		newRepositories := make([]repository.Repository, 0, len(repositories))
		for _, repo := range repositories {
			if repository.ContainsRepository(cfg.Repositories, repo.Path) {
				knownCount++
				continue
			}

			newCount++
			newRepositories = append(newRepositories, repo)
		}

		if newCount > 0 || workspaceAdded {
			cfg.Repositories = append(cfg.Repositories, newRepositories...)
			if err := config.Save(cfg); err != nil {
				return err
			}
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Scanning %s...\n\n", directory)
		for _, repo := range repositories {
			fmt.Fprintf(out, "✓ %s\n  %s\n\n", repo.Name, repo.Path)
		}
		fmt.Fprintf(out, "Found: %d\nNew:   %d\nKnown: %d\n", len(repositories), newCount, knownCount)

		return nil
	},
}

func containsString(values []string, value string) bool {
	cleanValue, err := filepath.Abs(value)
	if err != nil {
		cleanValue = filepath.Clean(value)
	}
	for _, existing := range values {
		cleanExisting, err := filepath.Abs(existing)
		if err != nil {
			cleanExisting = filepath.Clean(existing)
		}
		if cleanExisting == cleanValue {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
