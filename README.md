# Git Workspace Manager

`gw` is a small Windows-friendly Go CLI for managing many local Git repositories from one place.

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

## JSON

Read-only commands support machine-readable output:

```powershell
gw status --json > repositories.json
gw list --json
gw info ForFunPr --json
```

## Roadmap

The detailed implementation roadmap is in [`git_workspace_manager_plan.md`](git_workspace_manager_plan.md). The initial releases focus on reliable repository discovery, status, remote information, synchronization, and safe maintenance. Future work may add dashboards, tags, favorites, and broader development-environment features.
