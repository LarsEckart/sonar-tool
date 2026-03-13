package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const AppName = "sonar-issues"

func ConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configHome, AppName, "config.json"), nil
}
