package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"gitworkspacefun/internal/gitcli"

	"github.com/spf13/cobra"
)

var infoJSON bool

type repositoryInfo struct {
	Name       string           `json:"name"`
	Path       string           `json:"path"`
	Branch     string           `json:"branch"`
	Dirty      bool             `json:"dirty"`
	Remote     string           `json:"remote,omitempty"`
	Ahead      int              `json:"ahead"`
	Behind     int              `json:"behind"`
	Sync       gitcli.SyncState `json:"sync"`
	LastCommit *gitcli.Commit   `json:"last_commit,omitempty"`
}

var infoCmd = &cobra.Command{
	Use:   "info <repository>",
	Short: "Show detailed information about a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := loadRepositories()
		if err != nil {
			return err
		}
		repositories, err := selectRepositories(cfg, args[0])
		if err != nil {
			return err
		}
		repo := repositories[0]
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		record, err := readStatus(ctx, repo)
		if err != nil {
			return err
		}
		info := repositoryInfo{
			Name: record.Name, Path: record.Path, Branch: record.Branch,
			Dirty: record.Dirty, Remote: record.Remote, Ahead: record.Ahead,
			Behind: record.Behind, Sync: record.Sync,
		}
		commit, commitErr := gitcli.GetLatestCommit(ctx, repo.Path)
		if commitErr == nil {
			info.LastCommit = &commit
		}
		if infoJSON {
			return writeJSON(cmd.OutOrStdout(), info)
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "Repository")
		fmt.Fprintln(writer, "────────────")
		fmt.Fprintf(writer, "Name\t%s\nPath\t%s\n\n", info.Name, info.Path)
		fmt.Fprintln(writer, "Git")
		fmt.Fprintf(writer, "Branch\t%s\nStatus\t%s\n\n", info.Branch, worktreeState(info.Dirty))
		fmt.Fprintln(writer, "Remote")
		fmt.Fprintf(writer, "origin\t%s\n\n", info.Remote)
		fmt.Fprintln(writer, "Sync")
		fmt.Fprintf(writer, "State\t%s\nAhead\t%d\nBehind\t%d\n", info.Sync, info.Ahead, info.Behind)
		if info.LastCommit != nil {
			fmt.Fprintf(writer, "\nLast commit\nHash\t%s\nMessage\t%s\nAuthor\t%s\nDate\t%s\n", info.LastCommit.Hash, info.LastCommit.Message, info.LastCommit.Author, info.LastCommit.Timestamp.Format(time.RFC3339))
		}
		return writer.Flush()
	},
}

func worktreeState(dirty bool) string {
	if dirty {
		return "dirty"
	}
	return "clean"
}

func init() {
	infoCmd.Flags().BoolVar(&infoJSON, "json", false, "print JSON output")
	rootCmd.AddCommand(infoCmd)
}
