package cmd

import (
	"fmt"
	"path/filepath"

	"gitworkspacefun/internal/config"

	"github.com/spf13/cobra"
)

var workspaceAddCmd = &cobra.Command{
	Use:   "add <directory>",
	Short: "Add a workspace directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		if !containsString(cfg.Workspaces, path) {
			cfg.Workspaces = append(cfg.Workspaces, filepath.Clean(path))
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added workspace %s.\n", path)
		return nil
	},
}

var workspaceRemoveCmd = &cobra.Command{
	Use:   "remove <directory>",
	Short: "Remove a workspace directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		kept := cfg.Workspaces[:0]
		removed := false
		for _, workspacePath := range cfg.Workspaces {
			if filepath.Clean(workspacePath) == filepath.Clean(path) {
				removed = true
				continue
			}
			kept = append(kept, workspacePath)
		}
		if !removed {
			return fmt.Errorf("workspace %q not found", args[0])
		}
		cfg.Workspaces = kept
		return config.Save(cfg)
	},
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured workspaces",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		for _, workspacePath := range cfg.Workspaces {
			fmt.Fprintln(cmd.OutOrStdout(), workspacePath)
		}
		return nil
	},
}

var workspaceCmd = &cobra.Command{Use: "workspace", Short: "Manage workspace directories"}

func init() {
	workspaceCmd.AddCommand(workspaceAddCmd, workspaceRemoveCmd, workspaceListCmd)
	rootCmd.AddCommand(workspaceCmd)
}
