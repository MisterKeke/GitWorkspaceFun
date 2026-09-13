package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ignoredDirs = map[string]bool{
	".cache":       true,
	"bin":          true,
	"build":        true,
	"dist":         true,
	"node_modules": true,
	"obj":          true,
	"vendor":       true,
}

// Scan discovers Git repositories below root.
func Scan(root string) ([]Repository, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve scan directory: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat scan directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan path is not a directory: %s", root)
	}

	repositories := make([]Repository, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}

		gitPath := filepath.Join(path, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			repositories = append(repositories, Repository{
				Name: filepath.Base(path),
				Path: filepath.Clean(path),
			})
			return filepath.SkipDir
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect %s: %w", gitPath, err)
		}

		if filepath.Clean(path) != filepath.Clean(root) && ignoredDirs[strings.ToLower(entry.Name())] {
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan repositories: %w", err)
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Path < repositories[j].Path
	})

	return repositories, nil
}

// ContainsRepository reports whether path is already registered.
func ContainsRepository(repositories []Repository, path string) bool {
	normalizedPath := normalizePath(path)
	for _, repo := range repositories {
		if normalizePath(repo.Path) == normalizedPath {
			return true
		}
	}

	return false
}

func normalizePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}

	return filepath.Clean(path)
}
