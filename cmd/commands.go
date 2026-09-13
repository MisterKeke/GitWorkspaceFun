package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"
	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"
	"github.com/MisterKeke/GitWorkspaceFun/workspace"
	"github.com/spf13/cobra"
)

const version = "0.3.0"

func publicRepositories(values []repository.Repository) []workspace.Repository {
	result := make([]workspace.Repository, 0, len(values))
	for _, value := range values {
		result = append(result, workspace.Repository{Name: value.Name, Path: value.Path})
	}
	return result
}

func (a *app) repositories() (*config.Config, []workspace.Repository, error) {
	cfg, err := a.load()
	if err != nil {
		return nil, []workspace.Repository{}, err
	}
	return cfg, publicRepositories(cfg.Repositories), nil
}

func selectRepositories(cfg *config.Config, name string) ([]workspace.Repository, error) {
	values := cfg.Repositories
	if name == "" {
		return publicRepositories(values), nil
	}
	repo, err := repository.FindByName(values, name)
	if err != nil {
		return []workspace.Repository{}, err
	}
	return []workspace.Repository{{Name: repo.Name, Path: repo.Path}}, nil
}

func writeJSON(out interface{ Write([]byte) (int, error) }, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func containsString(values []string, value string) bool {
	cleanValue, err := filepath.Abs(value)
	if err != nil {
		cleanValue = filepath.Clean(value)
	}
	for _, existing := range values {
		cleanExisting, absErr := filepath.Abs(existing)
		if absErr != nil {
			cleanExisting = filepath.Clean(existing)
		}
		if cleanExisting == cleanValue {
			return true
		}
	}
	return false
}

func newInitCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "init", Short: "Initialize Git Workspace Manager", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		path, err := a.configFile()
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Git Workspace already initialized.")
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := config.SaveAt(path, config.Default()); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Created:")
		fmt.Fprintln(cmd.OutOrStdout(), filepath.Dir(path))
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	}}
}

func newScanCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "scan <directory>", Short: "Scan a directory for Git repositories", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		directory, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolve scan directory: %w", err)
		}
		repositories, err := a.manager.Scan(cmd.Context(), directory)
		if err != nil {
			return err
		}
		cfg, err := a.load()
		if err != nil {
			return err
		}
		workspaceAdded := false
		if !containsString(cfg.Workspaces, directory) {
			cfg.Workspaces = append(cfg.Workspaces, directory)
			workspaceAdded = true
		}
		newCount, knownCount := 0, 0
		newRepositories := make([]repository.Repository, 0, len(repositories))
		for _, repo := range repositories {
			if repository.ContainsRepository(cfg.Repositories, repo.Path) {
				knownCount++
				continue
			}
			newCount++
			newRepositories = append(newRepositories, repository.Repository{Name: repo.Name, Path: repo.Path})
		}
		if newCount > 0 || workspaceAdded {
			cfg.Repositories = append(cfg.Repositories, newRepositories...)
			if err := a.save(cfg); err != nil {
				return err
			}
		}
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Scanning %s...\n\n", directory)
		for _, repo := range repositories {
			fmt.Fprintf(out, "✓ %s\n  %s\n\n", repo.Name, repo.Path)
		}
		fmt.Fprintf(out, "Found: %d\nNew:   %d\nKnown: %d\n", len(repositories), newCount, knownCount)
		return nil
	}}
}

func newRescanCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "rescan", Short: "Rescan all configured workspaces", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		if len(cfg.Workspaces) == 0 {
			return fmt.Errorf("no workspaces configured; run gw scan <directory> first")
		}
		found := make([]repository.Repository, 0)
		for _, root := range cfg.Workspaces {
			if _, statErr := os.Stat(root); os.IsNotExist(statErr) {
				continue
			}
			values, scanErr := a.manager.Scan(cmd.Context(), root)
			if scanErr != nil {
				return fmt.Errorf("rescan %s: %w", root, scanErr)
			}
			for _, value := range values {
				candidate := repository.Repository{Name: value.Name, Path: value.Path}
				if !repository.ContainsRepository(found, candidate.Path) {
					found = append(found, candidate)
				}
			}
		}
		for _, existing := range cfg.Repositories {
			if _, statErr := os.Stat(existing.Path); os.IsNotExist(statErr) && !repository.ContainsRepository(found, existing.Path) {
				found = append(found, existing)
			}
		}
		cfg.Repositories = found
		if err := a.save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Rescanned %d repositories across %d workspaces.\n", len(found), len(cfg.Workspaces))
		return nil
	}}
}

func newListCommand(a *app) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "list", Short: "List registered Git repositories", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(cmd.OutOrStdout(), cfg.Repositories)
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tPATH")
		for _, repo := range cfg.Repositories {
			fmt.Fprintf(writer, "%s\t%s\n", repo.Name, repo.Path)
		}
		return writer.Flush()
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	return command
}

func newStatusCommand(a *app) *cobra.Command {
	var jsonOutput, onlyDirty, onlyClean, onlyAhead, onlyBehind bool
	var branch string
	command := &cobra.Command{Use: "status [repository]", Short: "Show the worktree and sync status of repositories", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := a.repositories()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		repos, err := selectRepositories(cfg, name)
		if err != nil {
			return err
		}
		results, statusErr := a.manager.CollectStatus(cmd.Context(), repos)
		filtered := make([]workspace.StatusResult, 0, len(results))
		for _, result := range results {
			if result.Status == nil {
				continue
			}
			status := result.Status
			if onlyDirty && !status.Changes.IsDirty() || onlyClean && status.Changes.IsDirty() || onlyAhead && status.Sync.Ahead == 0 || onlyBehind && status.Sync.Behind == 0 || branch != "" && status.Branch.Name != branch {
				continue
			}
			filtered = append(filtered, result)
		}
		if jsonOutput {
			if err := writeJSON(cmd.OutOrStdout(), filtered); err != nil {
				return err
			}
		} else {
			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(writer, "REPOSITORY\tBRANCH\tSTATUS\tSYNC\tAHEAD\tBEHIND")
			for _, result := range filtered {
				status := result.Status
				state := "clean"
				if status.Changes.IsDirty() {
					state = "dirty"
				}
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%d\t%d\n", status.Repository.Name, status.Branch.Name, state, status.Sync.State, status.Sync.Ahead, status.Sync.Behind)
			}
			if err := writer.Flush(); err != nil {
				return err
			}
		}
		if statusErr != nil {
			return statusErr
		}
		return nil
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	command.Flags().BoolVar(&onlyDirty, "dirty", false, "show only dirty repositories")
	command.Flags().BoolVar(&onlyClean, "clean", false, "show only clean repositories")
	command.Flags().BoolVar(&onlyAhead, "ahead", false, "show repositories ahead of upstream")
	command.Flags().BoolVar(&onlyBehind, "behind", false, "show repositories behind upstream")
	command.Flags().StringVar(&branch, "branch", "", "filter by branch")
	return command
}

func newBranchCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "branch", Short: "Show the current branch for each repository", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "REPOSITORY\tBRANCH")
		for _, repo := range repos {
			status, statusErr := a.manager.Status(cmd.Context(), repo)
			if statusErr != nil {
				return fmt.Errorf("get branch for %s: %w", repo.Name, statusErr)
			}
			fmt.Fprintf(writer, "%s\t%s\n", repo.Name, status.Branch.Name)
		}
		return writer.Flush()
	}}
}

func newDirtyCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "dirty [repository]", Short: "Show uncommitted changes", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := a.repositories()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		repos, err := selectRepositories(cfg, name)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		dirty := 0
		for _, repo := range repos {
			status, statusErr := a.manager.Status(cmd.Context(), repo)
			if statusErr != nil {
				return fmt.Errorf("get status for %s: %w", repo.Name, statusErr)
			}
			if !status.Changes.IsDirty() {
				continue
			}
			dirty++
			fmt.Fprintln(out, repo.Name)
			for _, line := range status.Changes.Lines {
				fmt.Fprintf(out, "  %s\n", strings.TrimLeft(line, " "))
			}
			fmt.Fprintln(out)
		}
		if dirty == 0 {
			if name != "" {
				fmt.Fprintf(out, "%s is clean.\n", repos[0].Name)
			} else {
				fmt.Fprintln(out, "No dirty repositories.")
			}
		}
		return nil
	}}
}

func newInfoCommand(a *app) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "info <repository>", Short: "Show detailed information about a repository", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := a.repositories()
		if err != nil {
			return err
		}
		repos, err := selectRepositories(cfg, args[0])
		if err != nil {
			return err
		}
		status, err := a.manager.Status(cmd.Context(), repos[0])
		if err != nil {
			return err
		}
		commit, commitErr := a.manager.LatestCommit(cmd.Context(), repos[0])
		var last *workspace.Commit
		if commitErr == nil {
			last = &commit
		}
		value := struct {
			Name       string              `json:"name"`
			Path       string              `json:"path"`
			Branch     string              `json:"branch"`
			Dirty      bool                `json:"dirty"`
			Remote     string              `json:"remote,omitempty"`
			Ahead      int                 `json:"ahead"`
			Behind     int                 `json:"behind"`
			Sync       workspace.SyncState `json:"sync"`
			LastCommit *workspace.Commit   `json:"last_commit,omitempty"`
		}{repos[0].Name, repos[0].Path, status.Branch.Name, status.Changes.IsDirty(), status.Remote, status.Sync.Ahead, status.Sync.Behind, status.Sync.State, last}
		if jsonOutput {
			return writeJSON(cmd.OutOrStdout(), value)
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "Repository")
		fmt.Fprintln(writer, "────────────")
		fmt.Fprintf(writer, "Name\t%s\nPath\t%s\n\n", value.Name, value.Path)
		fmt.Fprintln(writer, "Git")
		state := "clean"
		if value.Dirty {
			state = "dirty"
		}
		fmt.Fprintf(writer, "Branch\t%s\nStatus\t%s\n\n", value.Branch, state)
		fmt.Fprintln(writer, "Remote")
		fmt.Fprintf(writer, "origin\t%s\n\n", value.Remote)
		fmt.Fprintln(writer, "Sync")
		fmt.Fprintf(writer, "State\t%s\nAhead\t%d\nBehind\t%d\n", value.Sync, value.Ahead, value.Behind)
		if last != nil {
			fmt.Fprintf(writer, "\nLast commit\nHash\t%s\nMessage\t%s\nAuthor\t%s\nDate\t%s\n", last.Hash, last.Message, last.Author, last.Timestamp.Format(time.RFC3339))
		}
		return writer.Flush()
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	return command
}

func newLogCommand(a *app) *cobra.Command {
	var limit int
	var jsonOutput bool
	command := &cobra.Command{Use: "log <repository>", Short: "Show recent commits", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := a.repositories()
		if err != nil {
			return err
		}
		repos, err := selectRepositories(cfg, args[0])
		if err != nil {
			return err
		}
		commits, err := a.manager.History(cmd.Context(), repos[0], limit)
		if err != nil {
			return fmt.Errorf("get log for %s: %w", repos[0].Name, err)
		}
		if jsonOutput {
			return writeJSON(cmd.OutOrStdout(), commits)
		}
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		for _, commit := range commits {
			size := 7
			if len(commit.Hash) < size {
				size = len(commit.Hash)
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", commit.Hash[:size], commit.Message, commit.Author)
		}
		return writer.Flush()
	}}
	command.Flags().IntVar(&limit, "limit", workspace.DefaultHistoryLimit, "number of commits to show")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output")
	return command
}

func managerWithWorkers(a *app, workers int) *workspace.Manager {
	if workers <= 0 {
		workers = workspace.DefaultWorkers
	}
	return workspace.New(workspace.Options{Workers: workers, Logger: cliLogger{app: a}})
}

func printOutcomes(cmd *cobra.Command, outcomes []workspace.RepositoryOutcome, pull bool) int {
	failed := 0
	for _, value := range outcomes {
		if pull {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", value.Repository.Name)
			switch value.Outcome {
			case workspace.OutcomeUpdated:
				fmt.Fprintln(cmd.OutOrStdout(), "✓ updated")
			case workspace.OutcomeAlreadyUpToDate:
				fmt.Fprintln(cmd.OutOrStdout(), "✓ already up to date")
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "⚠ %s\n", value.Message)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		} else if value.Outcome == workspace.OutcomeFailed || value.Outcome == workspace.OutcomeCancelled {
			failed++
			fmt.Fprintf(cmd.ErrOrStderr(), "✗ %-20s %s\n", value.Repository.Name, value.Error)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", value.Repository.Name)
		}
	}
	return failed
}

func newFetchCommand(a *app) *cobra.Command {
	var workers int
	command := &cobra.Command{Use: "fetch", Short: "Fetch updates for all registered repositories", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		outcomes, actionErr := managerWithWorkers(a, workers).Fetch(cmd.Context(), repos)
		failed := printOutcomes(cmd, outcomes, false)
		if actionErr != nil {
			return actionErr
		}
		if failed > 0 {
			return fmt.Errorf("%d repository fetches failed", failed)
		}
		return nil
	}}
	command.Flags().IntVar(&workers, "workers", workspace.DefaultWorkers, "maximum concurrent fetches")
	return command
}

func newPullCommand(a *app) *cobra.Command {
	var workers int
	command := &cobra.Command{Use: "pull", Short: "Fast-forward clean repositories that are behind", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		outcomes, actionErr := managerWithWorkers(a, workers).Pull(cmd.Context(), repos)
		failed := printOutcomes(cmd, outcomes, true)
		if actionErr != nil {
			return actionErr
		}
		if failed > 0 {
			return fmt.Errorf("%d repository pulls failed", failed)
		}
		return nil
	}}
	command.Flags().IntVar(&workers, "workers", workspace.DefaultWorkers, "maximum concurrent pulls")
	return command
}

func newSyncCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "sync", Short: "Fetch and safely pull repositories", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		result, syncErr := managerWithWorkers(a, workspace.DefaultWorkers).Sync(cmd.Context(), repos)
		printOutcomes(cmd, result.Fetch, false)
		printOutcomes(cmd, result.Pull, true)
		if syncErr != nil {
			return syncErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Sync complete.")
		return nil
	}}
}

func newOpenCommand(a *app) *cobra.Command {
	var explorer, remote bool
	command := &cobra.Command{Use: "open <repository>", Short: "Open a repository in an editor, Explorer, or its remote", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := a.repositories()
		if err != nil {
			return err
		}
		repos, err := selectRepositories(cfg, args[0])
		if err != nil {
			return err
		}
		repo := repos[0]
		program, argument := cfg.Editor, repo.Path
		if explorer {
			program = "explorer.exe"
		} else if remote {
			rawRemote, webErr := a.manager.RemoteWebURL(cmd.Context(), repo, "origin")
			if webErr != nil {
				return webErr
			}
			program, argument = "explorer.exe", rawRemote
		}
		if program == "" {
			return fmt.Errorf("no editor configured")
		}
		if err := exec.Command(program, argument).Start(); err != nil {
			return fmt.Errorf("open %s: %w", repo.Name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Opened %s.\n", repo.Name)
		return nil
	}}
	command.Flags().BoolVar(&explorer, "explorer", false, "open in Windows Explorer")
	command.Flags().BoolVar(&remote, "remote", false, "open the origin remote in a browser")
	return command
}

func newPruneCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "prune", Short: "Remove missing repositories from the configuration", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		kept := cfg.Repositories[:0]
		removed := 0
		for _, repo := range cfg.Repositories {
			if _, statErr := os.Stat(repo.Path); os.IsNotExist(statErr) {
				removed++
				continue
			}
			kept = append(kept, repo)
		}
		cfg.Repositories = kept
		if err := a.save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed %d missing repositories.\n", removed)
		return nil
	}}
}

func newDoctorCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Check the Git Workspace Manager environment", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Environment")
		fmt.Fprintln(out, "───────────")
		env, gitErr := a.manager.DetectGit(cmd.Context())
		if gitErr == nil {
			fmt.Fprintf(out, "✓ Git installed\n  %s\n", env.Version)
		} else {
			fmt.Fprintln(out, "✗ Git not found")
		}
		path, pathErr := a.configFile()
		if pathErr != nil {
			return pathErr
		}
		cfg, loadErr := config.LoadAt(path)
		if loadErr != nil {
			fmt.Fprintf(out, "✗ Config unreadable: %v\n", loadErr)
			return nil
		}
		fmt.Fprintf(out, "✓ Config readable\n  %s\n", path)
		missing, remotes := 0, 0
		for _, repo := range publicRepositories(cfg.Repositories) {
			if _, statErr := os.Stat(repo.Path); os.IsNotExist(statErr) {
				missing++
				continue
			}
			if status, statusErr := a.manager.Status(cmd.Context(), repo); statusErr == nil && status.Remote != "" {
				remotes++
			}
		}
		fmt.Fprintf(out, "✓ %d repositories registered\n", len(cfg.Repositories))
		if missing > 0 {
			fmt.Fprintf(out, "⚠ %d repository paths missing\n", missing)
		} else {
			fmt.Fprintln(out, "✓ Repository paths present")
		}
		if editor, editorErr := exec.LookPath(cfg.Editor); editorErr == nil {
			fmt.Fprintf(out, "✓ Editor found\n  %s\n", editor)
		} else {
			fmt.Fprintf(out, "⚠ Editor %q not found\n", cfg.Editor)
		}
		fmt.Fprintf(out, "✓ Git remotes\n  %d / %d\n", remotes, len(cfg.Repositories))
		return nil
	}}
}

func newStaleCommand(a *app) *cobra.Command {
	var days int
	command := &cobra.Command{Use: "stale", Short: "Show repositories whose latest commit is old", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		values, err := a.manager.Forgotten(cmd.Context(), repos, workspace.ForgottenOptions{OlderThan: time.Duration(days) * 24 * time.Hour})
		if err != nil {
			return err
		}
		for _, value := range values {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", value.Repository.Name, value.Commit.Timestamp.Format("2006-01-02"))
		}
		return nil
	}}
	command.Flags().IntVar(&days, "days", 30, "minimum age in days")
	return command
}

func newForgottenCommand(a *app) *cobra.Command {
	var dirty bool
	command := &cobra.Command{Use: "forgotten", Short: "Show repositories with old commits", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		values, err := a.manager.Forgotten(cmd.Context(), repos, workspace.ForgottenOptions{DirtyOnly: dirty})
		if err != nil {
			return err
		}
		for _, value := range values {
			state := "clean"
			if value.Dirty {
				state = "dirty"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", value.Repository.Name, value.Commit.Timestamp.Format("2006-01-02"), state)
		}
		return nil
	}}
	command.Flags().BoolVar(&dirty, "dirty", false, "show only dirty repositories")
	return command
}

func newDashboardCommand(a *app) *cobra.Command {
	return &cobra.Command{Use: "dashboard", Short: "Show an aggregate workspace dashboard", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, repos, err := a.repositories()
		if err != nil {
			return err
		}
		results, statusErr := a.manager.CollectStatus(cmd.Context(), repos)
		branches := make(map[string]int)
		dirty, ahead, behind, diverged := 0, 0, 0, 0
		for _, result := range results {
			if result.Status == nil {
				continue
			}
			status := result.Status
			branches[status.Branch.Name]++
			if status.Changes.IsDirty() {
				dirty++
			}
			if status.Sync.Ahead > 0 {
				ahead++
			}
			if status.Sync.Behind > 0 {
				behind++
			}
			if status.Sync.State == workspace.SyncDiverged {
				diverged++
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Git Workspace")
		fmt.Fprintln(cmd.OutOrStdout(), "─────────────")
		fmt.Fprintf(cmd.OutOrStdout(), "Repositories\t%d\n\n", len(results))
		fmt.Fprintln(cmd.OutOrStdout(), "Status")
		fmt.Fprintf(cmd.OutOrStdout(), "Clean\t%d\nDirty\t%d\n\n", len(results)-dirty, dirty)
		fmt.Fprintln(cmd.OutOrStdout(), "Remote")
		fmt.Fprintf(cmd.OutOrStdout(), "Ahead\t%d\nBehind\t%d\nDiverged\t%d\n\n", ahead, behind, diverged)
		fmt.Fprintln(cmd.OutOrStdout(), "Branches")
		names := make([]string, 0, len(branches))
		for name := range branches {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\n", name, branches[name])
		}
		return statusErr
	}}
}

func newWorkspaceCommand(a *app) *cobra.Command {
	add := &cobra.Command{Use: "add <directory>", Short: "Add a workspace directory", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		if !containsString(cfg.Workspaces, path) {
			cfg.Workspaces = append(cfg.Workspaces, filepath.Clean(path))
		}
		if err := a.save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added workspace %s.\n", path)
		return nil
	}}
	remove := &cobra.Command{Use: "remove <directory>", Short: "Remove a workspace directory", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		kept := cfg.Workspaces[:0]
		removed := false
		for _, value := range cfg.Workspaces {
			if filepath.Clean(value) == filepath.Clean(path) {
				removed = true
				continue
			}
			kept = append(kept, value)
		}
		if !removed {
			return fmt.Errorf("workspace %q not found", args[0])
		}
		cfg.Workspaces = kept
		return a.save(cfg)
	}}
	list := &cobra.Command{Use: "list", Short: "List configured workspaces", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		for _, value := range cfg.Workspaces {
			fmt.Fprintln(cmd.OutOrStdout(), value)
		}
		return nil
	}}
	command := &cobra.Command{Use: "workspace", Short: "Manage workspace directories"}
	command.AddCommand(add, remove, list)
	return command
}

func newConfigCommand(a *app) *cobra.Command {
	get := &cobra.Command{Use: "get <key>", Short: "Get a configuration value", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		if args[0] != "editor" {
			return fmt.Errorf("unknown configuration key %q", args[0])
		}
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Editor)
		return nil
	}}
	set := &cobra.Command{Use: "set <key> <value>", Short: "Set a configuration value", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		if args[0] != "editor" {
			return fmt.Errorf("unknown configuration key %q", args[0])
		}
		cfg.Editor = args[1]
		return a.save(cfg)
	}}
	list := &cobra.Command{Use: "list", Short: "List configuration values", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := a.load()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "editor\t%s\nworkspaces\t%d\nrepositories\t%d\n", cfg.Editor, len(cfg.Workspaces), len(cfg.Repositories))
		return nil
	}}
	command := &cobra.Command{Use: "config", Short: "Manage configuration values"}
	command.AddCommand(get, set, list)
	return command
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{Use: "version", Short: "Show version information", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "gw %s\n%s %s/%s\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	}}
}
