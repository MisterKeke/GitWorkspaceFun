package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	statusJSON   bool
	statusDirty  bool
	statusClean  bool
	statusAhead  bool
	statusBehind bool
	statusBranch string
)

var statusCmd = &cobra.Command{
	Use:   "status [repository]",
	Short: "Show the worktree and sync status of repositories",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, repositories, err := loadRepositories()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		repositories, err = selectRepositories(cfg, name)
		if err != nil {
			return err
		}
		records, errorsFound := collectStatusRecords(cmd.Context(), repositories)
		if err := joinErrors(errorsFound); err != nil {
			return err
		}
		filtered := records[:0]
		for _, record := range records {
			if statusDirty && !record.Dirty || statusClean && record.Dirty ||
				statusAhead && record.Ahead == 0 || statusBehind && record.Behind == 0 ||
				statusBranch != "" && record.Branch != statusBranch {
				continue
			}
			filtered = append(filtered, record)
		}
		if statusJSON {
			return writeJSON(cmd.OutOrStdout(), filtered)
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "REPOSITORY\tBRANCH\tSTATUS\tSYNC\tAHEAD\tBEHIND")
		for _, record := range filtered {
			worktree := "clean"
			if record.Dirty {
				worktree = "dirty"
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%d\t%d\n", record.Name, record.Branch, worktree, record.Sync, record.Ahead, record.Behind)
		}
		return writer.Flush()
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "print JSON output")
	statusCmd.Flags().BoolVar(&statusDirty, "dirty", false, "show only dirty repositories")
	statusCmd.Flags().BoolVar(&statusClean, "clean", false, "show only clean repositories")
	statusCmd.Flags().BoolVar(&statusAhead, "ahead", false, "show repositories ahead of upstream")
	statusCmd.Flags().BoolVar(&statusBehind, "behind", false, "show repositories behind upstream")
	statusCmd.Flags().StringVar(&statusBranch, "branch", "", "filter by branch")
	rootCmd.AddCommand(statusCmd)
}
