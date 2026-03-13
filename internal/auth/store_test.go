package auth

import (
	"os"
	"path/filepath"
	"testing"
)

type fakeSecretStore struct {
	values map[string]string
}

func newFakeSecretStore() *fakeSecretStore {
	return &fakeSecretStore{values: map[string]string{}}
}

func (s *fakeSecretStore) Set(service, user, secret string) error {
	s.values[service+"|"+user] = secret
	return nil
}

func (s *fakeSecretStore) Get(service, user string) (string, error) {
	value, ok := s.values[service+"|"+user]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (s *fakeSecretStore) Delete(service, user string) error {
	delete(s.values, service+"|"+user)
	return nil
}

func (s *fakeSecretStore) DeleteAll(service string) error {
	for key := range s.values {
		if len(key) >= len(service)+1 && key[:len(service)+1] == service+"|" {
			delete(s.values, key)
		}
	}
	return nil
}

func TestStoreLoginResolveAndLogout(t *testing.T) {
	t.Setenv("SONAR_TOKEN", "")
	t.Setenv("SONAR_TOOL_TOKEN", "")
	t.Setenv("SONAR_ORG", "")
	t.Setenv("SONAR_HOST_URL", "")

	configPath := filepath.Join(t.TempDir(), "config.json")
	secrets := newFakeSecretStore()
	store := NewStore(configPath, secrets)

	if err := store.Login("https://sonarcloud.io", "my-org", "secret-token"); err != nil {
		t.Fatalf("login: %v", err)
	}

	resolved, err := store.Resolve("", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Host != "https://sonarcloud.io" {
		t.Fatalf("resolved host = %q, want %q", resolved.Host, "https://sonarcloud.io")
	}
	if resolved.Org != "my-org" {
		t.Fatalf("resolved org = %q, want %q", resolved.Org, "my-org")
	}
	if resolved.Token != "secret-token" {
		t.Fatalf("resolved token = %q, want %q", resolved.Token, "secret-token")
	}
	if resolved.TokenSource != "keychain" {
		t.Fatalf("resolved token source = %q, want keychain", resolved.TokenSource)
	}

	profile, err := store.CurrentProfile()
	if err != nil {
		t.Fatalf("current profile: %v", err)
	}
	if profile.Org != "my-org" {
		t.Fatalf("current profile org = %q, want %q", profile.Org, "my-org")
	}

	if err := store.Logout("https://sonarcloud.io", "my-org"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := store.CurrentProfile(); err == nil {
		t.Fatal("expected current profile error after logout")
	}
}

func TestStoreResolvePrefersKeychainOverEnvironmentToken(t *testing.T) {
	t.Setenv("SONAR_TOOL_TOKEN", "")
	t.Setenv("SONAR_ORG", "")
	t.Setenv("SONAR_HOST_URL", "")

	configPath := filepath.Join(t.TempDir(), "config.json")
	secrets := newFakeSecretStore()
	store := NewStore(configPath, secrets)

	if err := store.Login("https://sonarcloud.io", "my-org", "stored-token"); err != nil {
		t.Fatalf("login: %v", err)
	}

	t.Setenv("SONAR_TOKEN", "env-token")
	resolved, err := store.Resolve("", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Token != "stored-token" {
		t.Fatalf("resolved token = %q, want %q", resolved.Token, "stored-token")
	}
	if resolved.TokenSource != "keychain" {
		t.Fatalf("resolved token source = %q, want keychain", resolved.TokenSource)
	}
}

func TestStoreResolveFallsBackToEnvironmentToken(t *testing.T) {
	t.Setenv("SONAR_ORG", "env-org")
	t.Setenv("SONAR_HOST_URL", "")
	t.Setenv("SONAR_TOKEN", "env-token")
	t.Setenv("SONAR_TOOL_TOKEN", "")

	configPath := filepath.Join(t.TempDir(), "config.json")
	secrets := newFakeSecretStore()
	store := NewStore(configPath, secrets)

	resolved, err := store.Resolve("", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Token != "env-token" {
		t.Fatalf("resolved token = %q, want %q", resolved.Token, "env-token")
	}
	if resolved.TokenSource != "SONAR_TOKEN" {
		t.Fatalf("resolved token source = %q, want SONAR_TOKEN", resolved.TokenSource)
	}
	if resolved.Org != "env-org" {
		t.Fatalf("resolved org = %q, want %q", resolved.Org, "env-org")
	}
}

func TestStoreLogoutAllClearsConfig(t *testing.T) {
	t.Setenv("SONAR_TOKEN", "")
	t.Setenv("SONAR_TOOL_TOKEN", "")
	t.Setenv("SONAR_ORG", "")
	t.Setenv("SONAR_HOST_URL", "")

	configPath := filepath.Join(t.TempDir(), "config.json")
	secrets := newFakeSecretStore()
	store := NewStore(configPath, secrets)

	if err := store.Login("https://sonarcloud.io", "one", "token-one"); err != nil {
		t.Fatalf("login one: %v", err)
	}
	if err := store.Login("https://sonarcloud.io", "two", "token-two"); err != nil {
		t.Fatalf("login two: %v", err)
	}

	if err := store.LogoutAll(); err != nil {
		t.Fatalf("logout all: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ActiveProfile != "" {
		t.Fatalf("active profile = %q, want empty", cfg.ActiveProfile)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("profiles length = %d, want 0", len(cfg.Profiles))
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file missing after logout all: %v", err)
	}
}
