package auth

import (
	"errors"
	"strings"

	keyring "github.com/zalando/go-keyring"
)

const KeyringService = "sonar-issues"

var ErrSecretNotFound = errors.New("secret not found")

type SecretStore interface {
	Set(service, user, secret string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
	DeleteAll(service string) error
}

type OSSecretStore struct{}

func (OSSecretStore) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

func (OSSecretStore) Get(service, user string) (string, error) {
	secret, err := keyring.Get(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrSecretNotFound
	}
	return secret, err
}

func (OSSecretStore) Delete(service, user string) error {
	err := keyring.Delete(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrSecretNotFound
	}
	return err
}

func (OSSecretStore) DeleteAll(service string) error {
	err := keyring.DeleteAll(service)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func TokenAccount(host, org string) string {
	host = strings.TrimSpace(host)
	org = strings.TrimSpace(org)
	return "token:" + host + ":" + org
}
