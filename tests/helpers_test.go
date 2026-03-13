package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	buildOnce  sync.Once
	binaryPath string
	buildErr   error
)

type cliResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type fixtureServer struct {
	server *httptest.Server
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine current file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		root := moduleRoot(t)
		binDir, err := os.MkdirTemp("", "sonar-issues-testbin-")
		if err != nil {
			buildErr = fmt.Errorf("create temp build dir: %w", err)
			return
		}
		binaryPath = filepath.Join(binDir, "sonar-issues.testbin")
		cmd := exec.Command("go", "build", "-o", binaryPath, ".")
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("build binary: %w\n%s", err, output)
		}
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return binaryPath
}

func runCLI(t *testing.T, env map[string]string, args ...string) cliResult {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), buildBinary(t), args...)
	cmd.Dir = moduleRoot(t)
	cmd.Env = append(os.Environ(), "TERM=dumb")
	for _, key := range []string{"SONAR_TOKEN", "SONAR_TOOL_TOKEN", "SONAR_ORG", "SONAR_HOST_URL", "XDG_CONFIG_HOME"} {
		cmd.Env = append(cmd.Env, key+"=")
	}
	if env == nil {
		env = map[string]string{}
	}
	if _, ok := env["XDG_CONFIG_HOME"]; !ok {
		env["XDG_CONFIG_HOME"] = t.TempDir()
	}
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	stdout, err := cmd.Output()
	result := cliResult{Stdout: string(stdout), ExitCode: 0}
	if err == nil {
		return result
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run command %v: %v", args, err)
	}
	result.Stderr = string(exitErr.Stderr)
	result.ExitCode = exitErr.ExitCode()
	return result
}

func newFixtureServer(t *testing.T) *fixtureServer {
	t.Helper()
	fixturesDir := filepath.Join(moduleRoot(t), "tests", "testdata")
	server := &fixtureServer{}
	server.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
			t.Fatalf("authorization header = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/authentication/validate":
			serveFixture(t, w, filepath.Join(fixturesDir, "auth_validate.json"))
		case "/api/organizations/search":
			if got, want := r.URL.Query().Get("organizations"), "example-org"; got != want {
				t.Fatalf("organizations query = %q, want %q", got, want)
			}
			serveFixture(t, w, filepath.Join(fixturesDir, "organizations_search.json"))
		case "/api/components/search_projects":
			if got, want := r.URL.Query().Get("organization"), "example-org"; got != want {
				t.Fatalf("organization query = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("p"), "1"; got != want {
				t.Fatalf("projects page query = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("ps"), "5"; got != want {
				t.Fatalf("projects page size query = %q, want %q", got, want)
			}
			serveFixture(t, w, filepath.Join(fixturesDir, "projects_page_1.json"))
		case "/api/issues/search":
			if got, want := r.URL.Query().Get("organization"), "example-org"; got != want {
				t.Fatalf("organization query = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("componentKeys"), "project-001"; got != want {
				t.Fatalf("componentKeys query = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("p"), "1"; got != want {
				t.Fatalf("issues page query = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("ps"), "3"; got != want {
				t.Fatalf("issues page size query = %q, want %q", got, want)
			}
			serveFixture(t, w, filepath.Join(fixturesDir, "issues_page_1.json"))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	return server
}

func (s *fixtureServer) URL() string {
	return s.server.URL
}

func (s *fixtureServer) Close() {
	s.server.Close()
}

func serveFixture(t *testing.T, w http.ResponseWriter, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	_, err = w.Write(data)
	if err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func writeConfigFixture(t *testing.T, host, org string) string {
	t.Helper()
	configHome := t.TempDir()
	configDir := filepath.Join(configHome, "sonar-issues")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	payload := map[string]any{
		"active_profile": host + "|" + org,
		"profiles": map[string]map[string]string{
			host + "|" + org: {
				"host": host,
				"org":  org,
			},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal config fixture: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return configHome
}

func trimTrailingWhitespace(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
}
