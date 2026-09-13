package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"gitworkspacefun/internal/repository"
)

func marshal(cfg *Config) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("config must not be nil")
	}

	serializable := *cfg
	if serializable.Workspaces == nil {
		serializable.Workspaces = []string{}
	}
	if serializable.Repositories == nil {
		serializable.Repositories = []repository.Repository{}
	}

	data, err := json.MarshalIndent(&serializable, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(data, '\n'), nil
}

func Save(cfg *Config) error {
	data, err := marshal(cfg)
	if err != nil {
		return err
	}

	path, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
