package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigFilePathMigratesLegacyConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	legacyPath := filepath.Join(configHome, LegacyAppName, "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("create legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("config file path: %v", err)
	}

	wantPath := filepath.Join(configHome, AppName, "config.json")
	if configPath != wantPath {
		t.Fatalf("config path = %q, want %q", configPath, wantPath)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected migrated config at %s: %v", configPath, err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy config to be moved, stat err = %v", err)
	}
}
