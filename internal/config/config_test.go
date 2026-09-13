package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	SetPath(path)
	t.Cleanup(func() { overridePath = "" })
	want := &Config{
		Workspaces: []string{filepath.Dir(path)},
		Repositories: []repository.Repository{{
			Name: "example", Path: filepath.Dir(path),
		}},
		Editor: "cursor",
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save returned an error: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if got.Editor != want.Editor || len(got.Repositories) != 1 || got.Repositories[0].Name != "example" {
		t.Fatalf("Load returned %+v, want %+v", got, want)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	SetPath(path)
	t.Cleanup(func() { overridePath = "" })
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("Load should reject invalid JSON")
	}
}
