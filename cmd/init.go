package cmd

import (
	"fmt"
	"os"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Git Workspace Manager",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.Dir()
		if err != nil {
			return err
		}

		configPath, err := config.Path()
		if err != nil {
			return err
		}

		_, err = os.Stat(configPath)

		if err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Git Workspace already initialized.")
			return nil
		}

		if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		cfg := config.Default()

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Created:")
		fmt.Fprintln(cmd.OutOrStdout(), dir)
		fmt.Fprintln(cmd.OutOrStdout(), configPath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
