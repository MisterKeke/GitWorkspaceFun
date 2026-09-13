package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"
	"github.com/MisterKeke/GitWorkspaceFun/internal/gitcli"
	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"
	"github.com/MisterKeke/GitWorkspaceFun/internal/workspace"
)

type statusRecord struct {
	Name      string           `json:"name"`
	Path      string           `json:"path"`
	Branch    string           `json:"branch"`
	Dirty     bool             `json:"dirty"`
	Modified  int              `json:"modified"`
	Added     int              `json:"added"`
	Deleted   int              `json:"deleted"`
	Renamed   int              `json:"renamed"`
	Untracked int              `json:"untracked"`
	Ahead     int              `json:"ahead"`
	Behind    int              `json:"behind"`
	Sync      gitcli.SyncState `json:"sync"`
	Upstream  string           `json:"upstream,omitempty"`
	Remote    string           `json:"remote,omitempty"`
}

func loadRepositories() (*config.Config, []repository.Repository, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	return cfg, cfg.Repositories, nil
}

func selectRepositories(cfg *config.Config, name string) ([]repository.Repository, error) {
	if name == "" {
		return cfg.Repositories, nil
	}
	repo, err := repository.FindByName(cfg.Repositories, name)
	if err != nil {
		return nil, err
	}
	return []repository.Repository{*repo}, nil
}

func readStatus(ctx context.Context, repo repository.Repository) (statusRecord, error) {
	operationCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	branch, err := gitcli.GetBranch(operationCtx, repo.Path)
	if err != nil {
		return statusRecord{}, fmt.Errorf("get branch for %s: %w", repo.Name, err)
	}
	changes, err := gitcli.GetStatus(operationCtx, repo.Path)
	if err != nil {
		return statusRecord{}, fmt.Errorf("get status for %s: %w", repo.Name, err)
	}
	syncStatus, err := gitcli.GetSyncStatus(operationCtx, repo.Path)
	if err != nil {
		return statusRecord{}, fmt.Errorf("get sync status for %s: %w", repo.Name, err)
	}
	remote := ""
	if rawRemote, remoteErr := gitcli.GetRemoteURL(operationCtx, repo.Path, "origin"); remoteErr == nil {
		remote = gitcli.NormalizeRemoteURL(rawRemote)
	}
	return statusRecord{
		Name: repo.Name, Path: repo.Path, Branch: branch,
		Dirty: changes.IsDirty(), Modified: changes.Modified, Added: changes.Added,
		Deleted: changes.Deleted, Renamed: changes.Renamed, Untracked: changes.Untracked,
		Ahead: syncStatus.Ahead, Behind: syncStatus.Behind, Sync: syncStatus.State,
		Upstream: syncStatus.Upstream, Remote: remote,
	}, nil
}

func collectStatusRecords(ctx context.Context, repositories []repository.Repository) ([]statusRecord, []error) {
	records := make([]statusRecord, len(repositories))
	indices := make(map[string]int, len(repositories))
	for i, repo := range repositories {
		indices[repo.Path] = i
	}
	results := workspace.ForEach(repositories, 4, func(repo repository.Repository) error {
		record, err := readStatus(ctx, repo)
		if err == nil {
			records[indices[repo.Path]] = record
		}
		return err
	})
	errorsFound := make([]error, 0)
	for _, result := range results {
		if result.Error != nil {
			errorsFound = append(errorsFound, result.Error)
		}
	}
	return records, errorsFound
}

func writeJSON(out interface{ Write([]byte) (int, error) }, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func joinErrors(errorsFound []error) error {
	if len(errorsFound) == 0 {
		return nil
	}
	messages := make([]string, 0, len(errorsFound))
	for _, err := range errorsFound {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf("%s", strings.Join(messages, "; "))
}
