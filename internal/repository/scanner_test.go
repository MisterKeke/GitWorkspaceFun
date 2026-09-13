package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFindsRepositoriesAndSkipsNestedContent(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"repo1", "repo2", "normal"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "repo1", ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "repo2", ".git"), []byte("gitdir: elsewhere"), 0644); err != nil {
		t.Fatal(err)
	}
	repositories, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned an error: %v", err)
	}
	if len(repositories) != 2 {
		t.Fatalf("Scan found %d repositories, want 2", len(repositories))
	}
}
