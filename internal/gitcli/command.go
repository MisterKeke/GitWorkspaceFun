// Package gitcli provides small wrappers around the installed Git command-line tool.
package gitcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes Git with repoPath as its working repository.
func Run(ctx context.Context, repoPath string, args ...string) (string, error) {
	if ctx == nil {
		return "", errors.New("git command context must not be nil")
	}

	gitArgs := make([]string, 0, len(args)+2)
	gitArgs = append(gitArgs, "-C", repoPath)
	gitArgs = append(gitArgs, args...)
	command := exec.CommandContext(ctx, "git", gitArgs...)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	output, err := command.Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("run git command: %w", ctxErr)
		}

		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return "", fmt.Errorf("run git %s: %w: %s", strings.Join(args, " "), err, detail)
		}

		return "", fmt.Errorf("run git %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimRight(string(output), "\r\n"), nil
}
