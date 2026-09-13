// Package workspace provides a context-aware, embeddable API for inspecting
// and safely synchronizing local Git repositories.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MisterKeke/GitWorkspaceFun/internal/gitcli"
)

const (
	MaxWorkers          = 16
	DefaultWorkers      = 4
	MaxHistoryLimit     = 100
	DefaultHistoryLimit = 5
	MaxScanResults      = 10000
	DefaultForgottenAge = 30 * 24 * time.Hour
)

var (
	ErrGitNotFound       = errors.New("git executable not found")
	ErrNoUpstream        = errors.New("no upstream branch")
	ErrUnsupportedRemote = errors.New("unsupported remote URL")
	ErrScanLimit         = errors.New("repository scan result limit exceeded")
	ErrInvalidOptions    = errors.New("invalid workspace options")
)

// Repository identifies a local Git repository.
type Repository struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Environment describes the installed Git environment.
type Environment struct {
	GitPath string `json:"git_path"`
	Version string `json:"version"`
}

// Branch describes the current branch or detached HEAD.
type Branch struct {
	Name     string `json:"name"`
	Detached bool   `json:"detached"`
	Commit   string `json:"commit,omitempty"`
}

// Changes contains porcelain status counts. A combined state such as AM is
// counted once using precedence rename, add/copy, delete, then modified.
type Changes struct {
	Modified  int      `json:"modified"`
	Added     int      `json:"added"`
	Deleted   int      `json:"deleted"`
	Renamed   int      `json:"renamed"`
	Untracked int      `json:"untracked"`
	Lines     []string `json:"lines"`
}

func (c Changes) IsDirty() bool {
	return c.Modified > 0 || c.Added > 0 || c.Deleted > 0 || c.Renamed > 0 || c.Untracked > 0
}

// SyncState describes the relationship between HEAD and its upstream.
type SyncState string

const (
	SyncClean      SyncState = "synced"
	SyncAhead      SyncState = "ahead"
	SyncBehind     SyncState = "behind"
	SyncDiverged   SyncState = "diverged"
	SyncNoUpstream SyncState = "no-upstream"
)

type SyncStatus struct {
	Upstream string    `json:"upstream,omitempty"`
	Ahead    int       `json:"ahead"`
	Behind   int       `json:"behind"`
	State    SyncState `json:"state"`
}

// RepositoryStatus is a structured snapshot of one repository.
type RepositoryStatus struct {
	Repository Repository `json:"repository"`
	Branch     Branch     `json:"branch"`
	Changes    Changes    `json:"changes"`
	Sync       SyncStatus `json:"sync"`
	Remote     string     `json:"remote,omitempty"`
}

// StatusResult preserves a successful status when another repository fails.
type StatusResult struct {
	Repository Repository        `json:"repository"`
	Status     *RepositoryStatus `json:"status,omitempty"`
	Error      string            `json:"error,omitempty"`
	Diagnostic error             `json:"-"`
}

// Outcome is the classification of a fetch, pull, or synchronization action.
type Outcome string

const (
	OutcomeUpdated           Outcome = "updated"
	OutcomeSkippedDirty      Outcome = "skipped_dirty"
	OutcomeSkippedNoUpstream Outcome = "skipped_no_upstream"
	OutcomeSkippedDiverged   Outcome = "skipped_diverged"
	OutcomeAlreadyUpToDate   Outcome = "already_up_to_date"
	OutcomeFailed            Outcome = "failed"
	OutcomeCancelled         Outcome = "cancelled"
)

type RepositoryOutcome struct {
	Repository Repository `json:"repository"`
	Outcome    Outcome    `json:"outcome"`
	Message    string     `json:"message,omitempty"`
	Error      string     `json:"error,omitempty"`
	Diagnostic error      `json:"-"`
}

type SyncResult struct {
	Fetch []RepositoryOutcome `json:"fetch"`
	Pull  []RepositoryOutcome `json:"pull"`
}

type Commit struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

type ForgottenRepository struct {
	Repository Repository `json:"repository"`
	Commit     Commit     `json:"commit"`
	Dirty      bool       `json:"dirty"`
}

type ScanOptions struct{ MaxResults int }

type Options struct {
	Runner            Runner
	Logger            Logger
	Workers           int
	RepositoryTimeout time.Duration
	NetworkTimeout    time.Duration
	MaxHistory        int
	MaxScanResults    int
}

type Logger interface {
	Printf(format string, args ...any)
}

type LoggerFunc func(format string, args ...any)

func (f LoggerFunc) Printf(format string, args ...any) { f(format, args...) }

// Runner is the finite, operation-level seam used for deterministic tests.
// It deliberately exposes no arbitrary Git argument list.
type Runner interface {
	Detect(ctx context.Context) (Environment, error)
	Branch(ctx context.Context, path string) (Branch, error)
	Status(ctx context.Context, path string) (Changes, error)
	Sync(ctx context.Context, path string) (SyncStatus, error)
	RemoteURL(ctx context.Context, path, remote string) (string, error)
	LatestCommit(ctx context.Context, path string) (Commit, error)
	History(ctx context.Context, path string, limit int) ([]Commit, error)
	Fetch(ctx context.Context, path string) error
	PullFastForward(ctx context.Context, path string) error
}

type Manager struct {
	runner Runner
	logger Logger
	opts   Options
}

func New(opts Options) *Manager {
	if opts.Workers <= 0 {
		opts.Workers = DefaultWorkers
	}
	if opts.Workers > MaxWorkers {
		opts.Workers = MaxWorkers
	}
	if opts.RepositoryTimeout <= 0 {
		opts.RepositoryTimeout = 10 * time.Second
	}
	if opts.NetworkTimeout <= 0 {
		opts.NetworkTimeout = 2 * time.Minute
	}
	if opts.MaxHistory <= 0 {
		opts.MaxHistory = DefaultHistoryLimit
	}
	if opts.MaxHistory > MaxHistoryLimit {
		opts.MaxHistory = MaxHistoryLimit
	}
	if opts.MaxScanResults <= 0 {
		opts.MaxScanResults = MaxScanResults
	}
	if opts.MaxScanResults > MaxScanResults {
		opts.MaxScanResults = MaxScanResults
	}
	if opts.Runner == nil {
		opts.Runner = commandRunner{}
	}
	if opts.Logger == nil {
		opts.Logger = discardLogger{}
	}
	return &Manager{runner: opts.Runner, logger: opts.Logger, opts: opts}
}

func (m *Manager) DetectGit(ctx context.Context) (Environment, error) { return m.runner.Detect(ctx) }

func DetectGit(ctx context.Context) (Environment, error) { return New(Options{}).DetectGit(ctx) }

func (m *Manager) Scan(ctx context.Context, root string, options ...ScanOptions) ([]Repository, error) {
	if ctx == nil {
		return []Repository{}, fmt.Errorf("scan: %w", ErrInvalidOptions)
	}
	if err := ctx.Err(); err != nil {
		return []Repository{}, err
	}
	limit := m.opts.MaxScanResults
	if len(options) > 0 && options[0].MaxResults > 0 {
		limit = options[0].MaxResults
	}
	if limit > MaxScanResults {
		limit = MaxScanResults
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return []Repository{}, fmt.Errorf("resolve scan directory: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return []Repository{}, fmt.Errorf("stat scan directory: %w", err)
	}
	if !info.IsDir() {
		return []Repository{}, fmt.Errorf("scan path is not a directory: %s", absRoot)
	}
	ignored := map[string]bool{".cache": true, "bin": true, "build": true, "dist": true, "node_modules": true, "obj": true, "vendor": true}
	result := make([]Repository, 0)
	walkErr := filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		gitPath := filepath.Join(path, ".git")
		if _, statErr := os.Stat(gitPath); statErr == nil {
			if len(result) >= limit {
				return ErrScanLimit
			}
			result = append(result, Repository{Name: filepath.Base(path), Path: filepath.Clean(path)})
			return filepath.SkipDir
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("inspect %s: %w", gitPath, statErr)
		}
		if filepath.Clean(path) != filepath.Clean(absRoot) && ignored[strings.ToLower(entry.Name())] {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	if walkErr != nil {
		return result, fmt.Errorf("scan repositories: %w", walkErr)
	}
	return result, nil
}

func Scan(ctx context.Context, root string, options ...ScanOptions) ([]Repository, error) {
	return New(Options{}).Scan(ctx, root, options...)
}

func Inspect(ctx context.Context, repo Repository) (RepositoryStatus, error) {
	return New(Options{}).Status(ctx, repo)
}

func CollectStatus(ctx context.Context, repositories []Repository) ([]StatusResult, error) {
	return New(Options{}).CollectStatus(ctx, repositories)
}

func Fetch(ctx context.Context, repositories []Repository) ([]RepositoryOutcome, error) {
	return New(Options{}).Fetch(ctx, repositories)
}

func Pull(ctx context.Context, repositories []Repository) ([]RepositoryOutcome, error) {
	return New(Options{}).Pull(ctx, repositories)
}

func Sync(ctx context.Context, repositories []Repository) (SyncResult, error) {
	return New(Options{}).Sync(ctx, repositories)
}

func (m *Manager) Status(ctx context.Context, repo Repository) (RepositoryStatus, error) {
	if ctx == nil {
		return RepositoryStatus{}, fmt.Errorf("status: %w", ErrInvalidOptions)
	}
	opCtx, cancel := context.WithTimeout(ctx, m.opts.RepositoryTimeout)
	defer cancel()
	branch, err := m.runner.Branch(opCtx, repo.Path)
	if err != nil {
		return RepositoryStatus{}, m.operationError("branch", repo, err)
	}
	changes, err := m.runner.Status(opCtx, repo.Path)
	if err != nil {
		return RepositoryStatus{}, m.operationError("status", repo, err)
	}
	if changes.Lines == nil {
		changes.Lines = []string{}
	}
	syncStatus, err := m.runner.Sync(opCtx, repo.Path)
	if err != nil {
		if errors.Is(err, ErrNoUpstream) {
			syncStatus = SyncStatus{State: SyncNoUpstream}
		} else {
			return RepositoryStatus{}, m.operationError("sync status", repo, err)
		}
	}
	remote := ""
	if raw, remoteErr := m.runner.RemoteURL(opCtx, repo.Path, "origin"); remoteErr == nil {
		remote, _ = SanitizeRemoteURL(raw)
	}
	return RepositoryStatus{Repository: repo, Branch: branch, Changes: changes, Sync: syncStatus, Remote: remote}, nil
}

// RemoteWebURL resolves and sanitizes a repository remote before it is used
// as a browser URL by a caller.
func (m *Manager) RemoteWebURL(ctx context.Context, repo Repository, remote string) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("remote URL: %w", ErrInvalidOptions)
	}
	opCtx, cancel := context.WithTimeout(ctx, m.opts.RepositoryTimeout)
	defer cancel()
	raw, err := m.runner.RemoteURL(opCtx, repo.Path, remote)
	if err != nil {
		return "", m.operationError("remote URL", repo, err)
	}
	return RemoteWebURL(raw)
}

func (m *Manager) CollectStatus(ctx context.Context, repositories []Repository) ([]StatusResult, error) {
	results := make([]StatusResult, len(repositories))
	for i, repo := range repositories {
		results[i].Repository = repo
	}
	var mu sync.Mutex
	outcomes, err := m.forEach(ctx, repositories, func(index int, repo Repository) RepositoryOutcome {
		status, statusErr := m.Status(ctx, repo)
		if statusErr != nil {
			results[index].Error = safeError(statusErr)
			results[index].Diagnostic = statusErr
			return RepositoryOutcome{Repository: repo, Outcome: outcomeForError(statusErr), Error: safeError(statusErr), Diagnostic: statusErr}
		}
		mu.Lock()
		results[index].Status = &status
		mu.Unlock()
		return RepositoryOutcome{Repository: repo, Outcome: OutcomeUpdated}
	})
	if err != nil {
		return results, err
	}
	for i, outcome := range outcomes {
		if outcome.Outcome == OutcomeCancelled && results[i].Status == nil {
			results[i].Error = outcome.Error
			results[i].Diagnostic = outcome.Diagnostic
		}
	}
	return results, aggregateStatusErrors(results)
}

func (m *Manager) LatestCommit(ctx context.Context, repo Repository) (Commit, error) {
	if ctx == nil {
		return Commit{}, fmt.Errorf("latest commit: %w", ErrInvalidOptions)
	}
	opCtx, cancel := context.WithTimeout(ctx, m.opts.RepositoryTimeout)
	defer cancel()
	commit, err := m.runner.LatestCommit(opCtx, repo.Path)
	if err != nil {
		return Commit{}, m.operationError("latest commit", repo, err)
	}
	return commit, nil
}

func LatestCommit(ctx context.Context, repo Repository) (Commit, error) {
	return New(Options{}).LatestCommit(ctx, repo)
}

func (m *Manager) History(ctx context.Context, repo Repository, limit int) ([]Commit, error) {
	if ctx == nil {
		return []Commit{}, fmt.Errorf("history: %w", ErrInvalidOptions)
	}
	if limit <= 0 {
		limit = m.opts.MaxHistory
	}
	if limit > m.opts.MaxHistory {
		limit = m.opts.MaxHistory
	}
	opCtx, cancel := context.WithTimeout(ctx, m.opts.RepositoryTimeout)
	defer cancel()
	commits, err := m.runner.History(opCtx, repo.Path, limit)
	if commits == nil {
		commits = []Commit{}
	}
	if err != nil {
		return commits, m.operationError("history", repo, err)
	}
	return commits, nil
}

func History(ctx context.Context, repo Repository, limit int) ([]Commit, error) {
	return New(Options{}).History(ctx, repo, limit)
}

func (m *Manager) Fetch(ctx context.Context, repositories []Repository) ([]RepositoryOutcome, error) {
	return m.action(ctx, repositories, func(opCtx context.Context, repo Repository) RepositoryOutcome {
		err := m.runner.Fetch(opCtx, repo.Path)
		if err != nil {
			return m.failedOutcome(repo, err)
		}
		return RepositoryOutcome{Repository: repo, Outcome: OutcomeUpdated}
	}, true)
}

func (m *Manager) Pull(ctx context.Context, repositories []Repository) ([]RepositoryOutcome, error) {
	return m.action(ctx, repositories, func(opCtx context.Context, repo Repository) RepositoryOutcome {
		inspectCtx, cancel := context.WithTimeout(opCtx, m.opts.RepositoryTimeout)
		changes, err := m.runner.Status(inspectCtx, repo.Path)
		cancel()
		if err != nil {
			return m.failedOutcome(repo, err)
		}
		if changes.IsDirty() {
			return RepositoryOutcome{Repository: repo, Outcome: OutcomeSkippedDirty, Message: "working tree contains changes"}
		}
		inspectCtx, cancel = context.WithTimeout(opCtx, m.opts.RepositoryTimeout)
		syncStatus, err := m.runner.Sync(inspectCtx, repo.Path)
		cancel()
		if err != nil {
			if errors.Is(err, ErrNoUpstream) {
				return RepositoryOutcome{Repository: repo, Outcome: OutcomeSkippedNoUpstream, Message: "no upstream branch"}
			}
			return m.failedOutcome(repo, err)
		}
		switch syncStatus.State {
		case SyncNoUpstream:
			return RepositoryOutcome{Repository: repo, Outcome: OutcomeSkippedNoUpstream, Message: "no upstream branch"}
		case SyncDiverged:
			return RepositoryOutcome{Repository: repo, Outcome: OutcomeSkippedDiverged, Message: "branch has diverged"}
		case SyncClean, SyncAhead:
			return RepositoryOutcome{Repository: repo, Outcome: OutcomeAlreadyUpToDate}
		case SyncBehind:
			if err := m.runner.PullFastForward(opCtx, repo.Path); err != nil {
				return m.failedOutcome(repo, err)
			}
			return RepositoryOutcome{Repository: repo, Outcome: OutcomeUpdated}
		default:
			return m.failedOutcome(repo, fmt.Errorf("unknown sync state %q", syncStatus.State))
		}
	}, true)
}

func (m *Manager) Sync(ctx context.Context, repositories []Repository) (SyncResult, error) {
	result := SyncResult{Fetch: make([]RepositoryOutcome, len(repositories)), Pull: make([]RepositoryOutcome, len(repositories))}
	fetch, fetchErr := m.Fetch(ctx, repositories)
	result.Fetch = fetch
	eligible := make([]Repository, 0, len(repositories))
	for i, outcome := range fetch {
		if outcome.Outcome == OutcomeUpdated {
			eligible = append(eligible, repositories[i])
		} else if outcome.Outcome == OutcomeCancelled {
			result.Pull[i] = RepositoryOutcome{Repository: repositories[i], Outcome: OutcomeCancelled, Message: "cancelled"}
		} else {
			result.Pull[i] = RepositoryOutcome{Repository: repositories[i], Outcome: OutcomeFailed, Message: "pull not attempted because fetch failed", Error: outcome.Error, Diagnostic: outcome.Diagnostic}
		}
	}
	pull, pullErr := m.Pull(ctx, eligible)
	index := 0
	for i := range result.Pull {
		if result.Pull[i].Outcome == "" && index < len(pull) {
			result.Pull[i] = pull[index]
			index++
		}
	}
	return result, errors.Join(fetchErr, pullErr)
}

func (m *Manager) Forgotten(ctx context.Context, repositories []Repository, options ...ForgottenOptions) ([]ForgottenRepository, error) {
	if ctx == nil {
		return []ForgottenRepository{}, fmt.Errorf("forgotten: %w", ErrInvalidOptions)
	}
	opts := ForgottenOptions{OlderThan: DefaultForgottenAge, Now: time.Now}
	if len(options) > 0 {
		if options[0].OlderThan > 0 {
			opts.OlderThan = options[0].OlderThan
		}
		if options[0].Now != nil {
			opts.Now = options[0].Now
		}
		opts.DirtyOnly = options[0].DirtyOnly
	}
	cutoff := opts.Now().Add(-opts.OlderThan)
	result := make([]ForgottenRepository, 0)
	for _, repo := range repositories {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		commit, err := m.LatestCommit(ctx, repo)
		if err != nil {
			return result, err
		}
		opCtx, cancel := context.WithTimeout(ctx, m.opts.RepositoryTimeout)
		changes, err := m.runner.Status(opCtx, repo.Path)
		cancel()
		if err != nil {
			return result, m.operationError("status", repo, err)
		}
		if commit.Timestamp.Before(cutoff) && (!opts.DirtyOnly || changes.IsDirty()) {
			result = append(result, ForgottenRepository{Repository: repo, Commit: commit, Dirty: changes.IsDirty()})
		}
	}
	return result, nil
}

type ForgottenOptions struct {
	OlderThan time.Duration
	DirtyOnly bool
	Now       func() time.Time
}

func (m *Manager) action(ctx context.Context, repositories []Repository, fn func(context.Context, Repository) RepositoryOutcome, network bool) ([]RepositoryOutcome, error) {
	results := make([]RepositoryOutcome, len(repositories))
	for i, repo := range repositories {
		results[i].Repository = repo
	}
	processed, err := m.forEach(ctx, repositories, func(index int, repo Repository) RepositoryOutcome {
		timeout := m.opts.RepositoryTimeout
		if network {
			timeout = m.opts.NetworkTimeout
		}
		opCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		outcome := fn(opCtx, repo)
		results[index] = outcome
		return outcome
	})
	if err != nil {
		return results, err
	}
	results = processed
	return results, aggregateOutcomeErrors(results)
}

func (m *Manager) forEach(ctx context.Context, repositories []Repository, fn func(int, Repository) RepositoryOutcome) ([]RepositoryOutcome, error) {
	if ctx == nil {
		return []RepositoryOutcome{}, fmt.Errorf("operation: %w", ErrInvalidOptions)
	}
	if len(repositories) == 0 {
		return []RepositoryOutcome{}, nil
	}
	workers := m.opts.Workers
	if workers > len(repositories) {
		workers = len(repositories)
	}
	results := make([]RepositoryOutcome, len(repositories))
	jobs := make(chan int)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				results[index] = fn(index, repositories[index])
			}
		}()
	}
	next := 0
	for next < len(repositories) {
		if err := ctx.Err(); err != nil {
			for ; next < len(repositories); next++ {
				results[next] = RepositoryOutcome{Repository: repositories[next], Outcome: OutcomeCancelled, Message: "cancelled", Error: safeError(err), Diagnostic: err}
			}
			break
		}
		select {
		case jobs <- next:
			next++
		case <-ctx.Done():
			for ; next < len(repositories); next++ {
				results[next] = RepositoryOutcome{Repository: repositories[next], Outcome: OutcomeCancelled, Message: "cancelled", Error: safeError(ctx.Err()), Diagnostic: ctx.Err()}
			}
			close(jobs)
			group.Wait()
			return results, nil
		}
	}
	close(jobs)
	group.Wait()
	return results, nil
}

func (m *Manager) failedOutcome(repo Repository, err error) RepositoryOutcome {
	opErr := m.operationError("git", repo, err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return RepositoryOutcome{Repository: repo, Outcome: OutcomeCancelled, Message: "cancelled", Error: safeError(opErr), Diagnostic: opErr}
	}
	return RepositoryOutcome{Repository: repo, Outcome: OutcomeFailed, Error: safeError(opErr), Diagnostic: opErr}
}

func (m *Manager) operationError(op string, repo Repository, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s %s cancelled: %w", op, repo.Name, err)
	}
	m.logger.Printf("workspace: %s %s: %v", op, repo.Path, err)
	return &OperationError{Operation: op, Repository: repo, Err: err}
}

type OperationError struct {
	Operation  string
	Repository Repository
	Err        error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s for %s failed", e.Operation, e.Repository.Name)
}
func (e *OperationError) Unwrap() error { return e.Err }

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func outcomeForError(err error) Outcome {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return OutcomeCancelled
	}
	return OutcomeFailed
}
func aggregateOutcomeErrors(results []RepositoryOutcome) error {
	errs := make([]error, 0)
	for _, r := range results {
		if r.Outcome == OutcomeFailed && r.Diagnostic != nil {
			errs = append(errs, r.Diagnostic)
		}
	}
	return errors.Join(errs...)
}
func aggregateStatusErrors(results []StatusResult) error {
	errs := make([]error, 0)
	for _, r := range results {
		if r.Diagnostic != nil {
			errs = append(errs, r.Diagnostic)
		}
	}
	return errors.Join(errs...)
}

func SanitizeRemoteURL(remote string) (string, error) {
	parsed, err := parseRemote(remote)
	if err != nil {
		return "", err
	}
	path := strings.TrimSuffix(parsed.Path, ".git")
	return strings.TrimPrefix(parsed.Host+path, "/"), nil
}

// NormalizeRemoteURL returns a safe display form, or an empty string for an
// unsupported remote.
func NormalizeRemoteURL(remote string) string {
	value, err := SanitizeRemoteURL(remote)
	if err != nil {
		return ""
	}
	return value
}

func RemoteWebURL(remote string) (string, error) {
	parsed, err := parseRemote(remote)
	if err != nil {
		return "", err
	}
	parsed.Scheme = "https"
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimSuffix(parsed.Path, ".git")
	parsed.RawPath = ""
	return parsed.String(), nil
}

func parseRemote(remote string) (*url.URL, error) {
	remote = strings.TrimSpace(remote)
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@"), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, ErrUnsupportedRemote
		}
		path := parts[1]
		if cut := strings.IndexAny(path, "?#"); cut >= 0 {
			path = path[:cut]
		}
		return &url.URL{Scheme: "ssh", Host: parts[0], Path: "/" + path}, nil
	}
	parsed, err := url.Parse(remote)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http" && parsed.Scheme != "ssh") {
		return nil, ErrUnsupportedRemote
	}
	return parsed, nil
}

type discardLogger struct{}

func (discardLogger) Printf(string, ...any) {}

type commandRunner struct{}

func (r commandRunner) Detect(ctx context.Context) (Environment, error) {
	if err := ctx.Err(); err != nil {
		return Environment{}, err
	}
	path, err := exec.LookPath("git")
	if err != nil {
		return Environment{}, ErrGitNotFound
	}
	opCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(opCtx, path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return Environment{}, fmt.Errorf("detect git: %w", err)
	}
	return Environment{GitPath: path, Version: strings.TrimSpace(string(output))}, nil
}
func (r commandRunner) Branch(ctx context.Context, path string) (Branch, error) {
	value, err := gitcli.GetBranch(ctx, path)
	if err != nil {
		return Branch{}, err
	}
	if strings.HasPrefix(value, "detached@") {
		return Branch{Name: value, Detached: true, Commit: strings.TrimPrefix(value, "detached@")}, nil
	}
	return Branch{Name: value}, nil
}
func (r commandRunner) Status(ctx context.Context, path string) (Changes, error) {
	value, err := gitcli.GetStatus(ctx, path)
	if err != nil {
		return Changes{}, err
	}
	return Changes{Modified: value.Modified, Added: value.Added, Deleted: value.Deleted, Renamed: value.Renamed, Untracked: value.Untracked, Lines: append([]string{}, value.Lines...)}, nil
}
func (r commandRunner) Sync(ctx context.Context, path string) (SyncStatus, error) {
	value, err := gitcli.GetSyncStatus(ctx, path)
	if err != nil {
		if errors.Is(err, gitcli.ErrNoUpstream) {
			return SyncStatus{State: SyncNoUpstream}, nil
		}
		return SyncStatus{}, err
	}
	return SyncStatus{Upstream: value.Upstream, Ahead: value.Ahead, Behind: value.Behind, State: SyncState(value.State)}, nil
}
func (r commandRunner) RemoteURL(ctx context.Context, path, remote string) (string, error) {
	return gitcli.GetRemoteURL(ctx, path, remote)
}
func (r commandRunner) LatestCommit(ctx context.Context, path string) (Commit, error) {
	value, err := gitcli.GetLatestCommit(ctx, path)
	return Commit{Hash: value.Hash, Message: value.Message, Author: value.Author, Timestamp: value.Timestamp}, err
}
func (r commandRunner) History(ctx context.Context, path string, limit int) ([]Commit, error) {
	values, err := gitcli.GetLog(ctx, path, limit)
	result := make([]Commit, 0, len(values))
	for _, value := range values {
		result = append(result, Commit{Hash: value.Hash, Message: value.Message, Author: value.Author, Timestamp: value.Timestamp})
	}
	return result, err
}
func (r commandRunner) Fetch(ctx context.Context, path string) error { return gitcli.Fetch(ctx, path) }
func (r commandRunner) PullFastForward(ctx context.Context, path string) error {
	return gitcli.PullFastForward(ctx, path)
}

// These helpers keep parsing behavior available to package consumers without
// exposing the internal Git command wrapper.
func ParseChanges(output string) (Changes, error) {
	value, err := gitcli.ParseChanges(output)
	if err != nil {
		return Changes{}, err
	}
	return Changes{Modified: value.Modified, Added: value.Added, Deleted: value.Deleted, Renamed: value.Renamed, Untracked: value.Untracked, Lines: append([]string{}, value.Lines...)}, nil
}

func ParseSyncCounts(output string) (ahead, behind int, err error) {
	return gitcli.ParseSyncCounts(output)
}
