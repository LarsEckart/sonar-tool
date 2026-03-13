package auth

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/LarsEckart/sonar-tool/internal/domain"
)

var (
	ErrNoActiveProfile = errors.New("no active auth profile")
	ErrProfileNotFound = errors.New("auth profile not found")
)

type Store struct {
	configPath string
	secrets    SecretStore
}

type ActiveProfile struct {
	ID   string `json:"id"`
	Host string `json:"host"`
	Org  string `json:"org"`
}

type ResolvedAuth struct {
	Host        string `json:"host"`
	Org         string `json:"org,omitzero"`
	Token       string `json:"-"`
	TokenSource string `json:"token_source"`
}

func NewStore(configPath string, secrets SecretStore) *Store {
	return &Store{configPath: configPath, secrets: secrets}
}

func (s *Store) Login(host, org, token string) error {
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		return err
	}

	profileID := ProfileID(host, org)
	cfg.Profiles[profileID] = Profile{Host: host, Org: org}
	cfg.ActiveProfile = profileID

	account := TokenAccount(host, org)
	if err := s.secrets.Set(KeyringService, account, token); err != nil {
		return fmt.Errorf("store token in keychain: %w", err)
	}
	_ = s.secrets.Delete(LegacyKeyringService, account)
	if err := SaveConfig(s.configPath, cfg); err != nil {
		return err
	}

	return nil
}

func (s *Store) CurrentProfile() (ActiveProfile, error) {
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		return ActiveProfile{}, err
	}

	if cfg.ActiveProfile == "" {
		return ActiveProfile{}, ErrNoActiveProfile
	}

	profile, ok := cfg.Profiles[cfg.ActiveProfile]
	if !ok {
		return ActiveProfile{}, ErrNoActiveProfile
	}

	return ActiveProfile{ID: cfg.ActiveProfile, Host: profile.Host, Org: profile.Org}, nil
}

func (s *Store) Logout(host, org string) error {
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		return err
	}

	profileID := ProfileID(host, org)
	profile, ok := cfg.Profiles[profileID]
	if !ok {
		return ErrProfileNotFound
	}

	if err := s.deleteToken(profile.Host, profile.Org); err != nil {
		return err
	}

	delete(cfg.Profiles, profileID)
	if cfg.ActiveProfile == profileID {
		cfg.ActiveProfile = nextActiveProfileID(cfg.Profiles)
	}

	return SaveConfig(s.configPath, cfg)
}

func (s *Store) LogoutAll() error {
	if err := s.deleteAllTokens(); err != nil {
		return err
	}

	cfg := Config{Profiles: map[string]Profile{}}
	return SaveConfig(s.configPath, cfg)
}

func (s *Store) ResolveTargetProfile(flagHost, flagOrg string) (ActiveProfile, error) {
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		return ActiveProfile{}, err
	}

	active, _ := s.CurrentProfile()
	resolvedHost := strings.TrimSpace(flagHost)
	if resolvedHost == "" {
		resolvedHost = active.Host
	}
	if resolvedHost == "" {
		resolvedHost = domain.DefaultHost
	}

	resolvedHost, err = domain.NormalizeHost(resolvedHost)
	if err != nil {
		return ActiveProfile{}, err
	}

	resolvedOrg := strings.TrimSpace(flagOrg)
	if resolvedOrg == "" {
		resolvedOrg = active.Org
	}
	if resolvedOrg == "" {
		return ActiveProfile{}, ErrNoActiveProfile
	}

	profileID := ProfileID(resolvedHost, resolvedOrg)
	profile, ok := cfg.Profiles[profileID]
	if !ok {
		return ActiveProfile{}, ErrProfileNotFound
	}

	return ActiveProfile{ID: profileID, Host: profile.Host, Org: profile.Org}, nil
}

func (s *Store) Resolve(flagHost, flagOrg string) (ResolvedAuth, error) {
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		return ResolvedAuth{}, err
	}

	active := ActiveProfile{}
	if cfg.ActiveProfile != "" {
		if profile, ok := cfg.Profiles[cfg.ActiveProfile]; ok {
			active = ActiveProfile{ID: cfg.ActiveProfile, Host: profile.Host, Org: profile.Org}
		}
	}

	host, err := domain.NormalizeHost(cmp.Or(strings.TrimSpace(flagHost), strings.TrimSpace(os.Getenv("SONAR_HOST_URL")), active.Host, domain.DefaultHost))
	if err != nil {
		return ResolvedAuth{}, err
	}

	org := strings.TrimSpace(cmp.Or(strings.TrimSpace(flagOrg), strings.TrimSpace(os.Getenv("SONAR_ORG")), active.Org))

	if org != "" {
		profileID := ProfileID(host, org)
		if profile, ok := cfg.Profiles[profileID]; ok {
			token, err := s.tokenFromKeychain(profile.Host, profile.Org)
			if err != nil {
				if errors.Is(err, ErrSecretNotFound) {
					return ResolvedAuth{}, ErrProfileNotFound
				}
				return ResolvedAuth{}, err
			}

			return ResolvedAuth{Host: profile.Host, Org: profile.Org, Token: token, TokenSource: "keychain"}, nil
		}
	}

	if token := strings.TrimSpace(os.Getenv("SONAR_TOKEN")); token != "" {
		return ResolvedAuth{Host: host, Org: org, Token: token, TokenSource: "SONAR_TOKEN"}, nil
	}
	if token := strings.TrimSpace(os.Getenv("SONAR_TOOL_TOKEN")); token != "" {
		return ResolvedAuth{Host: host, Org: org, Token: token, TokenSource: "SONAR_TOOL_TOKEN"}, nil
	}

	if org == "" {
		return ResolvedAuth{}, ErrNoActiveProfile
	}

	return ResolvedAuth{}, ErrProfileNotFound
}

func (s *Store) tokenFromKeychain(host, org string) (string, error) {
	account := TokenAccount(host, org)

	token, err := s.secrets.Get(KeyringService, account)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, ErrSecretNotFound) {
		return "", fmt.Errorf("load token from keychain: %w", err)
	}

	legacyToken, legacyErr := s.secrets.Get(LegacyKeyringService, account)
	if legacyErr == nil {
		if setErr := s.secrets.Set(KeyringService, account, legacyToken); setErr != nil {
			return "", fmt.Errorf("migrate token to keychain: %w", setErr)
		}
		_ = s.secrets.Delete(LegacyKeyringService, account)
		return legacyToken, nil
	}
	if errors.Is(legacyErr, ErrSecretNotFound) {
		return "", ErrSecretNotFound
	}
	return "", fmt.Errorf("load token from keychain: %w", legacyErr)
}

func (s *Store) deleteToken(host, org string) error {
	account := TokenAccount(host, org)
	var deleteErrs []error

	if err := s.secrets.Delete(KeyringService, account); err != nil && !errors.Is(err, ErrSecretNotFound) {
		deleteErrs = append(deleteErrs, fmt.Errorf("remove token from keychain: %w", err))
	}
	if err := s.secrets.Delete(LegacyKeyringService, account); err != nil && !errors.Is(err, ErrSecretNotFound) {
		deleteErrs = append(deleteErrs, fmt.Errorf("remove token from legacy keychain: %w", err))
	}
	if len(deleteErrs) > 0 {
		return errors.Join(deleteErrs...)
	}

	return nil
}

func (s *Store) deleteAllTokens() error {
	var deleteErrs []error

	if err := s.secrets.DeleteAll(KeyringService); err != nil {
		deleteErrs = append(deleteErrs, fmt.Errorf("remove tokens from keychain: %w", err))
	}
	if err := s.secrets.DeleteAll(LegacyKeyringService); err != nil {
		deleteErrs = append(deleteErrs, fmt.Errorf("remove tokens from legacy keychain: %w", err))
	}
	if len(deleteErrs) > 0 {
		return errors.Join(deleteErrs...)
	}

	return nil
}

func nextActiveProfileID(profiles map[string]Profile) string {
	if len(profiles) == 0 {
		return ""
	}

	keys := slices.Sorted(maps.Keys(profiles))
	return keys[0]
}
