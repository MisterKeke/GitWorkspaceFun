package gitcli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Commit describes a Git commit returned by the history commands.
type Commit struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

// GetLatestCommit returns the newest commit in a repository.
func GetLatestCommit(ctx context.Context, repoPath string) (Commit, error) {
	output, err := Run(ctx, repoPath, "log", "-1", "--format=%H%x00%s%x00%an%x00%aI")
	if err != nil {
		return Commit{}, err
	}
	return parseCommit(output)
}

// GetLog returns up to limit recent commits.
func GetLog(ctx context.Context, repoPath string, limit int) ([]Commit, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("log limit must be positive")
	}
	output, err := Run(ctx, repoPath, "log", "-n", strconv.Itoa(limit), "--format=%H%x00%s%x00%an%x00%aI")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return []Commit{}, nil
	}
	fields := strings.Split(output, "\x00")
	if len(fields)%4 != 0 {
		return nil, fmt.Errorf("invalid git log output")
	}
	commits := make([]Commit, 0, len(fields)/4)
	for i := 0; i < len(fields); i += 4 {
		commit, err := parseCommit(strings.Join(fields[i:i+4], "\x00"))
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

func parseCommit(output string) (Commit, error) {
	fields := strings.SplitN(strings.TrimSpace(output), "\x00", 4)
	if len(fields) != 4 {
		return Commit{}, fmt.Errorf("invalid git commit output")
	}
	timestamp, err := time.Parse(time.RFC3339, fields[3])
	if err != nil {
		return Commit{}, fmt.Errorf("parse commit timestamp: %w", err)
	}
	return Commit{Hash: fields[0], Message: fields[1], Author: fields[2], Timestamp: timestamp}, nil
}
