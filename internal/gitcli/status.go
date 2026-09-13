package gitcli

import (
	"context"
	"fmt"
	"strings"
)

// Changes contains counts of the changes reported by Git.
type Changes struct {
	Modified  int
	Added     int
	Deleted   int
	Renamed   int
	Untracked int

	// Lines contains the non-empty porcelain-v1 lines that produced these
	// counts. It is used by commands that need to show individual changes.
	Lines []string
}

// IsDirty reports whether the repository contains any detected changes.
func (c Changes) IsDirty() bool {
	return c.Modified > 0 ||
		c.Added > 0 ||
		c.Deleted > 0 ||
		c.Renamed > 0 ||
		c.Untracked > 0
}

// GetStatus returns change counts from Git's porcelain-v1 status output.
func GetStatus(ctx context.Context, repoPath string) (Changes, error) {
	output, err := Run(ctx, repoPath, "status", "--porcelain=v1")
	if err != nil {
		return Changes{}, err
	}

	return ParseChanges(output)
}

// ParseChanges parses the two-character status codes emitted by porcelain v1.
func ParseChanges(output string) (Changes, error) {
	var changes Changes
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		if len(line) < 2 {
			return Changes{}, fmt.Errorf("invalid porcelain status line %q", line)
		}

		code := line[:2]
		switch {
		case code == "??":
			changes.Untracked++
		case code == "!!":
			// Ignored files are not included by default, but do not count them
			// as changes if a caller provides output with --ignored.
		case strings.Contains(code, "R"):
			changes.Renamed++
		case strings.Contains(code, "A") || strings.Contains(code, "C"):
			changes.Added++
		case strings.Contains(code, "D"):
			changes.Deleted++
		default:
			changes.Modified++
		}
		changes.Lines = append(changes.Lines, line)
	}

	return changes, nil
}
