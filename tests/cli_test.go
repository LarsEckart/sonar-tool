package tests

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthCurrentWithoutProfile(t *testing.T) {
	result := runCLI(t, nil, "auth", "current")
	if got, want := result.ExitCode, 3; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}
	if !strings.Contains(result.Stderr, "error: no active auth profile") {
		t.Fatalf("stderr = %q, want no active auth profile", result.Stderr)
	}
}

func TestAuthCurrentWithStoredProfile(t *testing.T) {
	server := newFixtureServer(t)
	defer server.Close()

	configHome := writeConfigFixture(t, server.URL(), "example-org")
	result := runCLI(t, map[string]string{"XDG_CONFIG_HOME": configHome}, "auth", "current")
	if got, want := result.ExitCode, 0; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}
	stdout := trimTrailingWhitespace(result.Stdout)
	if !strings.Contains(stdout, "host: "+server.URL()) {
		t.Fatalf("stdout = %q, want host line", stdout)
	}
	if !strings.Contains(stdout, "org:  example-org") {
		t.Fatalf("stdout = %q, want org line", stdout)
	}
}

func TestAuthCheckWithEnvironmentToken(t *testing.T) {
	server := newFixtureServer(t)
	defer server.Close()

	result := runCLI(t, map[string]string{
		"SONAR_TOKEN":    "test-token",
		"SONAR_ORG":      "example-org",
		"SONAR_HOST_URL": server.URL(),
	}, "auth", "check")
	if got, want := result.ExitCode, 0; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}
	if got := trimTrailingWhitespace(result.Stdout); got != "Authentication OK for "+server.URL()+" (org: example-org)" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestProjectsListJSON(t *testing.T) {
	server := newFixtureServer(t)
	defer server.Close()

	result := runCLI(t, map[string]string{
		"SONAR_TOKEN":    "test-token",
		"SONAR_ORG":      "example-org",
		"SONAR_HOST_URL": server.URL(),
	}, "projects", "list", "--limit", "5", "--json")
	if got, want := result.ExitCode, 0; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}

	var payload struct {
		Query struct {
			Org   string `json:"org"`
			Limit int    `json:"limit"`
		} `json:"query"`
		Paging struct {
			PageSize int `json:"page_size"`
		} `json:"paging"`
		Projects []struct {
			Key string `json:"key"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout:\n%s", err, result.Stdout)
	}
	if got, want := payload.Query.Org, "example-org"; got != want {
		t.Fatalf("query org = %q, want %q", got, want)
	}
	if got, want := payload.Query.Limit, 5; got != want {
		t.Fatalf("query limit = %d, want %d", got, want)
	}
	if got, want := payload.Paging.PageSize, 5; got != want {
		t.Fatalf("page size = %d, want %d", got, want)
	}
	if len(payload.Projects) != 5 {
		t.Fatalf("project count = %d, want 5", len(payload.Projects))
	}
	if got, want := payload.Projects[0].Key, "project-001"; got != want {
		t.Fatalf("first project key = %q, want %q", got, want)
	}
}

func TestIssuesListJSON(t *testing.T) {
	server := newFixtureServer(t)
	defer server.Close()

	result := runCLI(t, map[string]string{
		"SONAR_TOKEN":    "test-token",
		"SONAR_ORG":      "example-org",
		"SONAR_HOST_URL": server.URL(),
	}, "issues", "list", "--project", "project-001", "--limit", "3", "--json")
	if got, want := result.ExitCode, 0; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}

	var payload struct {
		Query struct {
			Org     string `json:"org"`
			Project string `json:"project"`
			Limit   int    `json:"limit"`
		} `json:"query"`
		Paging struct {
			PageSize int `json:"page_size"`
		} `json:"paging"`
		Issues []struct {
			Key       string `json:"key"`
			Component string `json:"component"`
		} `json:"issues"`
		Rules []struct {
			Key string `json:"key"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout:\n%s", err, result.Stdout)
	}
	if got, want := payload.Query.Org, "example-org"; got != want {
		t.Fatalf("query org = %q, want %q", got, want)
	}
	if got, want := payload.Query.Project, "project-001"; got != want {
		t.Fatalf("query project = %q, want %q", got, want)
	}
	if got, want := payload.Query.Limit, 3; got != want {
		t.Fatalf("query limit = %d, want %d", got, want)
	}
	if len(payload.Issues) != 3 {
		t.Fatalf("issue count = %d, want 3", len(payload.Issues))
	}
	if len(payload.Rules) == 0 {
		t.Fatal("expected at least one rule")
	}
	if got, want := payload.Issues[0].Key, "issue-001"; got != want {
		t.Fatalf("first issue key = %q, want %q", got, want)
	}
}

func TestIssuesListValidationError(t *testing.T) {
	result := runCLI(t, nil, "issues", "list", "--project", "project-001", "--limit", "0")
	if got, want := result.ExitCode, 2; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}
	if !strings.Contains(result.Stderr, "error: limit must be between 1 and 500") {
		t.Fatalf("stderr = %q", result.Stderr)
	}
}

func TestIssuesListConflictingSortFlags(t *testing.T) {
	result := runCLI(t, nil, "issues", "list", "--project", "project-001", "--asc", "--desc")
	if got, want := result.ExitCode, 2; got != want {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", got, want, result.Stderr)
	}
	if !strings.Contains(result.Stderr, "error: --asc and --desc cannot be used together") {
		t.Fatalf("stderr = %q", result.Stderr)
	}
}
