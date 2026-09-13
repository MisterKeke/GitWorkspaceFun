package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"gitworkspacefun/internal/config"
	"gitworkspacefun/internal/gitcli"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the Git Workspace Manager environment",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Environment")
		fmt.Fprintln(out, "───────────")
		if gitPath, err := exec.LookPath("git"); err == nil {
			version, versionErr := exec.Command("git", "--version").Output()
			if versionErr == nil {
				fmt.Fprintf(out, "✓ Git installed\n  %s", version)
			} else {
				fmt.Fprintf(out, "✓ Git installed\n  %s\n", gitPath)
			}
		} else {
			fmt.Fprintln(out, "✗ Git not found")
		}
		path, pathErr := config.Path()
		if pathErr != nil {
			return pathErr
		}
		cfg, loadErr := config.Load()
		if loadErr != nil {
			fmt.Fprintf(out, "✗ Config unreadable: %v\n", loadErr)
			return nil
		}
		fmt.Fprintf(out, "✓ Config readable\n  %s\n", path)
		missing := 0
		remotes := 0
		for _, repo := range cfg.Repositories {
			if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
				missing++
				continue
			}
			if _, err := gitcli.GetRemoteURL(cmd.Context(), repo.Path, "origin"); err == nil {
				remotes++
			}
		}
		fmt.Fprintf(out, "✓ %d repositories registered\n", len(cfg.Repositories))
		if missing > 0 {
			fmt.Fprintf(out, "⚠ %d repository paths missing\n", missing)
		} else {
			fmt.Fprintln(out, "✓ Repository paths present")
		}
		if editor, err := exec.LookPath(cfg.Editor); err == nil {
			fmt.Fprintf(out, "✓ Editor found\n  %s\n", editor)
		} else {
			fmt.Fprintf(out, "⚠ Editor %q not found\n", cfg.Editor)
		}
		fmt.Fprintf(out, "✓ Git remotes\n  %d / %d\n", remotes, len(cfg.Repositories))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
