package workspace

import (
	"errors"
	"testing"

	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"
)

func TestForEachProcessesAllRepositories(t *testing.T) {
	repositories := []repository.Repository{
		{Name: "one", Path: "one"},
		{Name: "two", Path: "two"},
		{Name: "three", Path: "three"},
	}
	results := ForEach(repositories, 2, func(repo repository.Repository) error {
		if repo.Name == "two" {
			return errors.New("expected failure")
		}
		return nil
	})
	if len(results) != len(repositories) {
		t.Fatalf("got %d results, want %d", len(results), len(repositories))
	}
	if results[0].Error != nil || results[1].Error == nil || results[2].Error != nil {
		t.Fatalf("unexpected results: %+v", results)
	}
}
