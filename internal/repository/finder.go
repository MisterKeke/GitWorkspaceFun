package repository

import (
	"fmt"
	"sort"
	"strings"
)

// FindByName returns the repository matching name, ignoring letter case.
// It rejects both missing and ambiguous names.
func FindByName(repositories []Repository, name string) (*Repository, error) {
	matches := make([]int, 0, 1)
	for i := range repositories {
		if strings.EqualFold(repositories[i].Name, name) {
			matches = append(matches, i)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("repository %q not found", name)
	case 1:
		return &repositories[matches[0]], nil
	default:
		paths := make([]string, 0, len(matches))
		for _, index := range matches {
			paths = append(paths, repositories[index].Path)
		}
		sort.Strings(paths)

		var message strings.Builder
		fmt.Fprintf(&message, "repository name %q is ambiguous:\n", name)
		for i, path := range paths {
			fmt.Fprintf(&message, "\n%d. %s", i+1, path)
		}

		return nil, fmt.Errorf("%s", message.String())
	}
}
