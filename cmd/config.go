package cmd

import (
	"fmt"

	"gitworkspacefun/internal/config"

	"github.com/spf13/cobra"
)

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "editor":
			fmt.Fprintln(cmd.OutOrStdout(), cfg.Editor)
		default:
			return fmt.Errorf("unknown configuration key %q", args[0])
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "editor":
			cfg.Editor = args[1]
		default:
			return fmt.Errorf("unknown configuration key %q", args[0])
		}
		return config.Save(cfg)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configuration values",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "editor\t%s\n", cfg.Editor)
		fmt.Fprintf(cmd.OutOrStdout(), "workspaces\t%d\n", len(cfg.Workspaces))
		fmt.Fprintf(cmd.OutOrStdout(), "repositories\t%d\n", len(cfg.Repositories))
		return nil
	},
}

var configCmd = &cobra.Command{Use: "config", Short: "Manage configuration values"}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd, configListCmd)
	rootCmd.AddCommand(configCmd)
}
