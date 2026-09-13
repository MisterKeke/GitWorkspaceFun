package cmd

import (
	"fmt"
	"os/exec"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"
	"github.com/MisterKeke/GitWorkspaceFun/internal/gitcli"

	"github.com/spf13/cobra"
)

var openExplorer bool
var openRemote bool

var openCmd = &cobra.Command{
	Use:   "open <repository>",
	Short: "Open a repository in an editor, Explorer, or its remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		repositories, err := selectRepositories(cfg, args[0])
		if err != nil {
			return err
		}
		repo := repositories[0]
		program := cfg.Editor
		argument := repo.Path
		if openExplorer {
			program = "explorer.exe"
		} else if openRemote {
			remote, err := gitcli.GetRemoteURL(cmd.Context(), repo.Path, "origin")
			if err != nil {
				return fmt.Errorf("get origin for %s: %w", repo.Name, err)
			}
			program = "explorer.exe"
			argument = gitcli.RemoteWebURL(remote)
		}
		if program == "" {
			return fmt.Errorf("no editor configured")
		}
		if err := exec.Command(program, argument).Start(); err != nil {
			return fmt.Errorf("open %s: %w", repo.Name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Opened %s.\n", repo.Name)
		return nil
	},
}

func init() {
	openCmd.Flags().BoolVar(&openExplorer, "explorer", false, "open in Windows Explorer")
	openCmd.Flags().BoolVar(&openRemote, "remote", false, "open the origin remote in a browser")
	rootCmd.AddCommand(openCmd)
}
