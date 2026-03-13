package domain

import (
	"fmt"
	"net/url"
	"strings"
)

const DefaultHost = "https://sonarcloud.io"

type OrganizationKey string

type ProjectKey string

func ParseOrganizationKey(raw string) (OrganizationKey, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("organization key cannot be empty")
	}
	return OrganizationKey(value), nil
}

func ParseProjectKey(raw string) (ProjectKey, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("project key cannot be empty")
	}
	return ProjectKey(value), nil
}

func NormalizeHost(raw string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		candidate = DefaultHost
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("parse host URL: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("host must include scheme and host, for example %s", DefaultHost)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("host must not include a path")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("host must not include query or fragment")
	}

	normalized := strings.TrimRight(parsed.Scheme+"://"+parsed.Host, "/")
	return normalized, nil
}
