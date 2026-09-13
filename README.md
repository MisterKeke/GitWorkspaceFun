# Git Workspace Manager

`gw` is a small Windows-friendly Go CLI for managing many local Git repositories from one place.

The reusable library is published by this module as `github.com/MisterKeke/GitWorkspaceFun/workspace`.

## Installation

```powershell
go build -o gw.exe .
```

Put `gw.exe` on `PATH`, or run it from the project directory.

## Quick start

```powershell
gw init
gw scan D:\Projects
gw list
gw status
gw dirty
```

The configuration is stored in `%USERPROFILE%\.gw\config.json`.

## Commands

- `gw scan <directory>` discovers and registers repositories.
- `gw rescan` refreshes repositories in all configured workspaces.
- `gw list [--json]` lists registered repositories.
- `gw status [repository] [--json]` shows worktree and upstream sync state.
- `gw branch` shows current branches.
- `gw dirty [repository]` shows porcelain changes.
- `gw info <repository> [--json]` shows remote, sync, and latest commit details.
- `gw log <repository> [--limit N]` shows recent commits.
- `gw fetch [--workers N]` fetches all remotes with bounded concurrency.
- `gw pull` fast-forwards only clean repositories that are behind.
- `gw sync` runs fetch followed by safe pull.
- `gw open <repository> [--explorer|--remote]` opens a repository.
- `gw prune` removes missing repository entries from the config.
- `gw doctor` checks Git, the config, repositories, remotes, and the editor.
- `gw stale --days N` and `gw forgotten [--dirty]` find old repositories.
- `gw workspace add|remove|list` manages scan roots.
- `gw config get|set|list` manages configuration values such as `editor`.
- `gw version` prints version and runtime information.

## Safety

`gw status` never fetches implicitly. `gw pull` skips dirty, diverged, and no-upstream repositories and uses `git pull --ff-only` for the remaining eligible repositories.

The library applies the same rules and returns structured per-repository outcomes. Pull outcomes are `updated`, `skipped_dirty`, `skipped_no_upstream`, `skipped_diverged`, `already_up_to_date`, `failed`, or `cancelled`. It never exits the process, launches an editor/browser, or accepts arbitrary Git arguments. Remote display and web URLs remove credentials, query strings, and fragments; unsupported schemes are rejected.

Scanning and concurrent operations are cancellable, use bounded workers, stop scheduling after cancellation, and preserve successful results when another repository fails. Defaults cap workers at 16, history at 100 commits, and scan results at 10,000 repositories. Porcelain records are counted once: for a combined state such as `AM`, the index/worktree pair counts as one added change (rename, add/copy, delete, then modified precedence).

## Public package

Consumers only need the public `workspace` package. The default manager invokes the installed `git` executable, while tests and embedders can inject a finite operation-level `workspace.Runner` and logger:

```go
import (
    "context"
    "time"

    "github.com/MisterKeke/GitWorkspaceFun/workspace"
)

ctx := context.Background()
manager := workspace.New(workspace.Options{
    Workers:           4,
    RepositoryTimeout: 10 * time.Second,
    NetworkTimeout:    2 * time.Minute,
})

repositories, err := manager.Scan(ctx, `D:\Projects`)
if err != nil {
    // Scan may still contain partial results when cancellation or a limit occurs.
}
statuses, err := manager.CollectStatus(ctx, repositories)
outcomes, err := manager.Pull(ctx, repositories)
syncResult, err := manager.Sync(ctx, repositories)
```

The API also exposes Git environment detection, branch/detached-HEAD information, change counts, upstream ahead/behind state, latest commits and bounded history, remote sanitization, fetch, fast-forward-only pull, and fetch-plus-safe-pull synchronization. Every filesystem or Git operation accepts a `context.Context`; cancellation and configured per-repository/network deadlines are honored.

`.gw/config.json` is a CLI concern. The library does not read it automatically and does not expose the CLI config-path override.

Git must be installed and available on `PATH`. The library follows normal Go module compatibility expectations: exported types and behavior are intended to remain stable within a major version; breaking API changes require a new major module version.

## JSON

Read-only commands support machine-readable output:

```powershell
gw status --json > repositories.json
gw list --json
gw info ForFunPr --json
```
