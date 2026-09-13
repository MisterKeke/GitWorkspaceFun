package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show an aggregate workspace dashboard",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, repositories, err := loadRepositories()
		if err != nil {
			return err
		}
		records, errorsFound := collectStatusRecords(cmd.Context(), repositories)
		if err := joinErrors(errorsFound); err != nil {
			return err
		}
		branches := make(map[string]int)
		dirty, ahead, behind, diverged := 0, 0, 0, 0
		for _, record := range records {
			branches[record.Branch]++
			if record.Dirty {
				dirty++
			}
			if record.Ahead > 0 {
				ahead++
			}
			if record.Behind > 0 {
				behind++
			}
			if record.Sync == "diverged" {
				diverged++
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Git Workspace")
		fmt.Fprintln(cmd.OutOrStdout(), "─────────────")
		fmt.Fprintf(cmd.OutOrStdout(), "Repositories\t%d\n\n", len(records))
		fmt.Fprintln(cmd.OutOrStdout(), "Status")
		fmt.Fprintf(cmd.OutOrStdout(), "Clean\t%d\nDirty\t%d\n\n", len(records)-dirty, dirty)
		fmt.Fprintln(cmd.OutOrStdout(), "Remote")
		fmt.Fprintf(cmd.OutOrStdout(), "Ahead\t%d\nBehind\t%d\nDiverged\t%d\n\n", ahead, behind, diverged)
		fmt.Fprintln(cmd.OutOrStdout(), "Branches")
		branchNames := make([]string, 0, len(branches))
		for branch := range branches {
			branchNames = append(branchNames, branch)
		}
		sort.Strings(branchNames)
		for _, branch := range branchNames {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\n", branch, branches[branch])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
