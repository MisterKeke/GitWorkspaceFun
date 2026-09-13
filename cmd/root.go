package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MisterKeke/GitWorkspaceFun/internal/config"
	"github.com/MisterKeke/GitWorkspaceFun/workspace"
	"github.com/spf13/cobra"
)

// Options configures a CLI command graph. ConfigPath is intentionally a CLI
// option; the public workspace package never reads .gw/config.json.
type Options struct {
	ConfigPath string
	Manager    *workspace.Manager
}

type app struct {
	configPath string
	manager    *workspace.Manager
	verbose    bool
}

// NewRootCommand returns a fresh command graph on every call.
func NewRootCommand(options ...Options) *cobra.Command {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}
	a := &app{configPath: opts.ConfigPath, manager: opts.Manager}
	if a.manager == nil {
		a.manager = workspace.New(workspace.Options{Logger: cliLogger{app: a}})
	}
	root := &cobra.Command{
		Use: "gw", Short: "Git Workspace Manager", Long: "Git Workspace Manager",
		SilenceUsage: true,
	}
	root.PersistentFlags().StringVar(&a.configPath, "config", a.configPath, "use an alternate configuration file")
	root.PersistentFlags().BoolVar(&a.verbose, "verbose", false, "enable verbose diagnostics")
	root.AddCommand(
		newInitCommand(a), newScanCommand(a), newRescanCommand(a), newListCommand(a),
		newStatusCommand(a), newBranchCommand(a), newDirtyCommand(a), newInfoCommand(a),
		newLogCommand(a), newFetchCommand(a), newPullCommand(a), newSyncCommand(a),
		newOpenCommand(a), newPruneCommand(a), newDoctorCommand(a), newStaleCommand(a),
		newForgottenCommand(a), newDashboardCommand(a), newWorkspaceCommand(a),
		newConfigCommand(a), newVersionCommand(),
	)
	return root
}

// Execute is the programmatic entry point. It returns command errors to the
// caller; only main is responsible for choosing a process exit code.
func Execute() error { return NewRootCommand().Execute() }

func (a *app) load() (*config.Config, error) {
	path, err := a.configFile()
	if err != nil {
		return nil, err
	}
	return config.LoadAt(path)
}

func (a *app) save(cfg *config.Config) error {
	path, err := a.configFile()
	if err != nil {
		return err
	}
	return config.SaveAt(path, cfg)
}

func (a *app) configFile() (string, error) {
	if a.configPath != "" {
		return filepath.Abs(a.configPath)
	}
	return config.Path()
}

type cliLogger struct{ app *app }

func (l cliLogger) Printf(format string, args ...any) {
	if l.app.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG "+format+"\n", args...)
	}
}

func ExecuteContext(ctx context.Context, args []string) error {
	root := NewRootCommand()
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}
