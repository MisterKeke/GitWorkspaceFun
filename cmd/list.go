package cmd

import (
	"fmt"
	"text/tabwriter"

	"gitworkspacefun/internal/config"

	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered Git repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if listJSON {
			return writeJSON(cmd.OutOrStdout(), cfg.Repositories)
		}

		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tPATH")
		for _, repo := range cfg.Repositories {
			fmt.Fprintf(writer, "%s\t%s\n", repo.Name, repo.Path)
		}

		return writer.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "print JSON output")
	rootCmd.AddCommand(listCmd)
}
