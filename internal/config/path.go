package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	AppName       = "sonar-tool"
	LegacyAppName = "sonar-issues"
)

func ConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	configPath := filepath.Join(configHome, AppName, "config.json")
	legacyConfigPath := filepath.Join(configHome, LegacyAppName, "config.json")
	if err := migrateLegacyConfig(configPath, legacyConfigPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func migrateLegacyConfig(configPath, legacyConfigPath string) error {
	if _, err := os.Stat(configPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat config: %w", err)
	}

	if _, err := os.Stat(legacyConfigPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat legacy config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Rename(legacyConfigPath, configPath); err != nil {
		return fmt.Errorf("migrate legacy config: %w", err)
	}
	_ = os.Remove(filepath.Dir(legacyConfigPath))
	return nil
}
