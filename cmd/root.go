package cmd

import (
	"fmt"
	"os"

	"gitworkspacefun/internal/config"
	"gitworkspacefun/internal/gitcli"

	"github.com/spf13/cobra"
)

var configFile string
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "gw",
	Short: "Git Workspace Manager",
	Long:  "Git Workspace Manager",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if configFile != "" {
			config.SetPath(configFile)
		}
		gitcli.SetVerbose(verbose)
		if verbose {
			fmt.Fprintln(cmd.ErrOrStderr(), "DEBUG verbose mode enabled")
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "use an alternate configuration file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose diagnostics")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
