package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch and safely pull repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := fetchCmd.RunE(fetchCmd, nil); err != nil {
			return err
		}
		if err := pullCmd.RunE(pullCmd, nil); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Sync complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
