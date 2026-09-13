package gitcli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrNoUpstream = errors.New("no upstream branch")

type SyncState string

const (
	SyncClean    SyncState = "synced"
	SyncAhead    SyncState = "ahead"
	SyncBehind   SyncState = "behind"
	SyncDiverged SyncState = "diverged"
	SyncNoRemote SyncState = "no-upstream"
)

type SyncStatus struct {
	Upstream string    `json:"upstream,omitempty"`
	Ahead    int       `json:"ahead"`
	Behind   int       `json:"behind"`
	State    SyncState `json:"state"`
}

func GetUpstream(ctx context.Context, repoPath string) (string, error) {
	upstream, err := Run(ctx, repoPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "upstream") {
			return "", ErrNoUpstream
		}
		return "", err
	}
	if upstream == "" {
		return "", ErrNoUpstream
	}
	return upstream, nil
}

func GetSyncStatus(ctx context.Context, repoPath string) (SyncStatus, error) {
	upstream, err := GetUpstream(ctx, repoPath)
	if errors.Is(err, ErrNoUpstream) {
		return SyncStatus{State: SyncNoRemote}, nil
	}
	if err != nil {
		return SyncStatus{}, err
	}
	output, err := Run(ctx, repoPath, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return SyncStatus{}, err
	}
	ahead, behind, err := ParseSyncCounts(output)
	if err != nil {
		return SyncStatus{}, err
	}
	state := SyncClean
	switch {
	case ahead > 0 && behind > 0:
		state = SyncDiverged
	case ahead > 0:
		state = SyncAhead
	case behind > 0:
		state = SyncBehind
	}
	return SyncStatus{Upstream: upstream, Ahead: ahead, Behind: behind, State: state}, nil
}

// ParseSyncCounts parses the two integers emitted by rev-list --count.
func ParseSyncCounts(output string) (ahead, behind int, err error) {
	counts := strings.Fields(output)
	if len(counts) != 2 {
		return 0, 0, fmt.Errorf("invalid ahead/behind output %q", output)
	}
	ahead, err = strconv.Atoi(counts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}
	behind, err = strconv.Atoi(counts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}
	return ahead, behind, nil
}

func Fetch(ctx context.Context, repoPath string) error {
	_, err := Run(ctx, repoPath, "fetch")
	return err
}

func PullFastForward(ctx context.Context, repoPath string) error {
	_, err := Run(ctx, repoPath, "pull", "--ff-only")
	return err
}
