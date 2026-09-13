package gitcli

import (
	"context"
	"fmt"
)

// GetBranch returns the current branch for a repository. Detached HEADs are
// represented by the short commit ID, for example detached@a782bc1.
func GetBranch(ctx context.Context, repoPath string) (string, error) {
	branch, err := Run(ctx, repoPath, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	if branch != "" {
		return branch, nil
	}

	commit, err := Run(ctx, repoPath, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve detached HEAD: %w", err)
	}

	return "detached@" + commit, nil
}
