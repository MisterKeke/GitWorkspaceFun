package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MisterKeke/GitWorkspaceFun/internal/repository"
)

// Ensure creates the default configuration when it does not exist.
func Ensure() error {
	path, err := Path()
	if err != nil {
		return err
	}
	return EnsureAt(path)
}

// EnsureAt creates the default configuration at an explicit path.
func EnsureAt(path string) error {

	dir := filepath.Dir(path)

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := marshal(Default())
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadAt(path)
}

// LoadAt loads a configuration from an explicit path.
func LoadAt(path string) (*Config, error) {
	if err := EnsureAt(path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Workspaces == nil {
		cfg.Workspaces = []string{}
	}
	if cfg.Repositories == nil {
		cfg.Repositories = []repository.Repository{}
	}
	if cfg.Editor == "" {
		cfg.Editor = Default().Editor
	}

	return cfg, nil
}
