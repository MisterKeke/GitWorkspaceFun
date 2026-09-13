package gitcli

import (
	"context"
	"net/url"
	"strings"
)

// GetRemoteURL returns the configured URL for a remote such as origin.
func GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error) {
	return Run(ctx, repoPath, "remote", "get-url", remote)
}

// NormalizeRemoteURL turns common SSH and HTTPS remote forms into a compact
// host/path representation suitable for table output.
func NormalizeRemoteURL(remote string) string {
	remote = strings.TrimSpace(remote)
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@"), ":", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[0]+"/"+parts[1], ".git")
		}
	}
	parsed, err := url.Parse(remote)
	if err == nil && parsed.Host != "" {
		return strings.TrimSuffix(strings.TrimPrefix(parsed.Host+parsed.Path, "/"), ".git")
	}
	return strings.TrimSuffix(remote, ".git")
}

// RemoteWebURL converts common GitHub/GitLab/Bitbucket remotes to a browser URL.
func RemoteWebURL(remote string) string {
	remote = strings.TrimSpace(remote)
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@"), ":", 2)
		if len(parts) == 2 {
			remote = "https://" + parts[0] + "/" + parts[1]
		}
	}
	if parsed, err := url.Parse(remote); err == nil && parsed.Host != "" {
		if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "ssh" {
			return ""
		}
		parsed.Scheme = "https"
		parsed.User = nil
		parsed.Path = strings.TrimSuffix(parsed.Path, ".git")
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String()
	}
	return ""
}
