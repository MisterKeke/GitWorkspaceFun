package config

import (
	"os"
	"path/filepath"

	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"
)

const directoryName = ".gw"
const fileName = "config.json"

var overridePath string

// Config is the persistent configuration for Git Workspace Manager.
type Config struct {
	Workspaces   []string                `json:"workspaces"`
	Repositories []repository.Repository `json:"repositories"`
	Editor       string                  `json:"editor,omitempty"`
}

// Dir returns the directory used to store Git Workspace Manager's files.
func Dir() (string, error) {
	if overridePath != "" {
		return filepath.Dir(overridePath), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, directoryName), nil
}

// Path returns the path to the configuration file.
func Path() (string, error) {
	if overridePath != "" {
		return overridePath, nil
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, fileName), nil
}

// SetPath overrides the default configuration path for the current process.
// It is intended for the root command's --config flag and tests.
func SetPath(path string) {
	if absolute, err := filepath.Abs(path); err == nil {
		overridePath = filepath.Clean(absolute)
	} else {
		overridePath = filepath.Clean(path)
	}
}

// Default returns a new empty configuration.
func Default() *Config {
	return &Config{
		Workspaces:   []string{},
		Repositories: []repository.Repository{},
		Editor:       "code",
	}
}
