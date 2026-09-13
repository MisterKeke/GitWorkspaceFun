package workspace

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeRunner struct {
	changes   map[string]Changes
	sync      map[string]SyncStatus
	statusErr map[string]error
	fetch     map[string]error
	pulls     int
}

func (f *fakeRunner) Detect(context.Context) (Environment, error) {
	return Environment{GitPath: "git", Version: "git version test"}, nil
}
func (f *fakeRunner) Branch(context.Context, string) (Branch, error) {
	return Branch{Name: "main"}, nil
}
func (f *fakeRunner) Status(_ context.Context, path string) (Changes, error) {
	return f.changes[path], f.statusErr[path]
}
func (f *fakeRunner) Sync(_ context.Context, path string) (SyncStatus, error) {
	return f.sync[path], nil
}
func (f *fakeRunner) RemoteURL(context.Context, string, string) (string, error) {
	return "git@github.com:owner/repo.git", nil
}
func (f *fakeRunner) LatestCommit(context.Context, string) (Commit, error) {
	return Commit{Hash: "abc", Timestamp: time.Unix(0, 0)}, nil
}
func (f *fakeRunner) History(context.Context, string, int) ([]Commit, error) { return []Commit{}, nil }
func (f *fakeRunner) Fetch(_ context.Context, path string) error             { return f.fetch[path] }
func (f *fakeRunner) PullFastForward(_ context.Context, _ string) error      { f.pulls++; return nil }

func TestParseChangesCountsCombinedStateOnce(t *testing.T) {
	got, err := ParseChanges("AM staged-and-modified.go\n?? new.go\n")
	if err != nil {
		t.Fatal(err)
	}
	want := Changes{Added: 1, Untracked: 1, Lines: []string{"AM staged-and-modified.go", "?? new.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRemoteSanitization(t *testing.T) {
	display, err := SanitizeRemoteURL("https://user:secret@example.com/owner/repo.git?token=bad#fragment")
	if err != nil || display != "example.com/owner/repo" {
		t.Fatalf("display=%q err=%v", display, err)
	}
	web, err := RemoteWebURL("git@github.com:owner/repo.git")
	if err != nil || web != "https://github.com/owner/repo" {
		t.Fatalf("web=%q err=%v", web, err)
	}
	if _, err := RemoteWebURL("ftp://example.com/owner/repo.git"); !errors.Is(err, ErrUnsupportedRemote) {
		t.Fatalf("expected unsupported remote, got %v", err)
	}
}

func TestPullClassifiesSafetySkipsAndPartialFailure(t *testing.T) {
	runner := &fakeRunner{
		changes: map[string]Changes{"dirty": {Modified: 1}},
		sync: map[string]SyncStatus{
			"no-upstream": {State: SyncNoUpstream},
			"diverged":    {State: SyncDiverged},
			"current":     {State: SyncClean},
			"behind":      {State: SyncBehind},
		},
		fetch: map[string]error{"failed": errors.New("network")},
	}
	manager := New(Options{Runner: runner, Workers: 2})
	repos := []Repository{{Name: "dirty", Path: "dirty"}, {Name: "no-upstream", Path: "no-upstream"}, {Name: "diverged", Path: "diverged"}, {Name: "current", Path: "current"}, {Name: "behind", Path: "behind"}}
	results, err := manager.Pull(context.Background(), repos)
	if err != nil {
		t.Fatal(err)
	}
	want := []Outcome{OutcomeSkippedDirty, OutcomeSkippedNoUpstream, OutcomeSkippedDiverged, OutcomeAlreadyUpToDate, OutcomeUpdated}
	for i := range want {
		if results[i].Outcome != want[i] {
			t.Fatalf("result %d=%q, want %q", i, results[i].Outcome, want[i])
		}
	}
	if runner.pulls != 1 {
		t.Fatalf("pulls=%d, want 1", runner.pulls)
	}
	fetchResults, err := manager.Fetch(context.Background(), append(repos, Repository{Name: "failed", Path: "failed"}))
	if err == nil || fetchResults[len(fetchResults)-1].Outcome != OutcomeFailed {
		t.Fatalf("partial fetch results=%+v err=%v", fetchResults, err)
	}
}

func TestCancelledWorkIsClassifiedAndNotScheduled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &fakeRunner{fetch: map[string]error{}}
	repos := []Repository{{Name: "one", Path: "one"}, {Name: "two", Path: "two"}}
	results, err := New(Options{Runner: runner}).Fetch(ctx, repos)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Outcome != OutcomeCancelled {
			t.Fatalf("got %+v", result)
		}
	}
}

func TestCollectStatusPreservesSuccessfulRecords(t *testing.T) {
	runner := &fakeRunner{statusErr: map[string]error{"bad": errors.New("status unavailable")}}
	repos := []Repository{{Name: "good", Path: "good"}, {Name: "bad", Path: "bad"}}
	results, err := New(Options{Runner: runner}).CollectStatus(context.Background(), repos)
	if err == nil {
		t.Fatal("expected partial status error")
	}
	if results[0].Status == nil || results[1].Status != nil || results[1].Error == "" {
		t.Fatalf("unexpected partial results: %+v", results)
	}
}

func TestSyncPreservesFetchFailureAndDoesNotPullIt(t *testing.T) {
	runner := &fakeRunner{
		sync:  map[string]SyncStatus{"bad": {State: SyncBehind}, "good": {State: SyncClean}},
		fetch: map[string]error{"bad": errors.New("fetch failed")},
	}
	repos := []Repository{{Name: "bad", Path: "bad"}, {Name: "good", Path: "good"}}
	result, err := New(Options{Runner: runner}).Sync(context.Background(), repos)
	if err == nil {
		t.Fatal("expected sync error")
	}
	if result.Fetch[0].Outcome != OutcomeFailed || result.Pull[0].Outcome != OutcomeFailed {
		t.Fatalf("unexpected failed sync classification: %+v", result)
	}
	if result.Pull[1].Outcome != OutcomeAlreadyUpToDate || runner.pulls != 0 {
		t.Fatalf("unexpected safe sync behavior: %+v pulls=%d", result, runner.pulls)
	}
}

func TestScanHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := New(Options{}).Scan(ctx, t.TempDir())
	if !errors.Is(err, context.Canceled) || result == nil {
		t.Fatalf("result=%v err=%v", result, err)
	}
}
