package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lars/sonar-tool/internal/auth"
	configpath "github.com/lars/sonar-tool/internal/config"
	"github.com/lars/sonar-tool/internal/domain"
	"github.com/lars/sonar-tool/internal/sonar"
	cli "github.com/urfave/cli/v3"
)

type OutputMode string

const (
	OutputModePlain    OutputMode = "plain"
	OutputModeJSON     OutputMode = "json"
	OutputModeMarkdown OutputMode = "markdown"
)

type CLIError struct {
	ExitCode int
	Message  string
	Hint     string
}

func (e *CLIError) Error() string {
	return e.Message
}

func usageError(message, hint string) error {
	return &CLIError{ExitCode: 2, Message: message, Hint: hint}
}

func runtimeError(message string, err error) error {
	if err == nil {
		return &CLIError{ExitCode: 1, Message: message}
	}
	return &CLIError{ExitCode: 1, Message: fmt.Sprintf("%s: %v", message, err)}
}

func authError(message, hint string) error {
	return &CLIError{ExitCode: 3, Message: message, Hint: hint}
}

func notFoundError(message string) error {
	return &CLIError{ExitCode: 4, Message: message}
}

func interruptedError() error {
	return &CLIError{ExitCode: 130, Message: "interrupted"}
}

func Interrupted() error {
	return interruptedError()
}

func ExitCode(err error) int {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.ExitCode
	}
	return 1
}

func RenderError(w io.Writer, err error) {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		if cliErr.Message != "" {
			_, _ = fmt.Fprintf(w, "error: %s\n", cliErr.Message)
		}
		if cliErr.Hint != "" {
			_, _ = fmt.Fprintf(w, "hint: %s\n", cliErr.Hint)
		}
		return
	}
	_, _ = fmt.Fprintf(w, "error: %v\n", err)
}

func outputMode(cmd *cli.Command) (OutputMode, error) {
	jsonMode := cmd.Bool("json")
	markdownMode := cmd.Bool("markdown")
	if jsonMode && markdownMode {
		return "", usageError("--json and --markdown cannot be used together", "choose one output mode")
	}
	if jsonMode {
		return OutputModeJSON, nil
	}
	if markdownMode {
		return OutputModeMarkdown, nil
	}
	return OutputModePlain, nil
}

func timeoutFromCommand(cmd *cli.Command) (time.Duration, error) {
	seconds := cmd.Int("timeout")
	if seconds <= 0 {
		return 0, usageError("invalid value for --timeout", "use a positive number of seconds")
	}
	return time.Duration(seconds) * time.Second, nil
}

func hostFromCommand(cmd *cli.Command) (string, error) {
	return domain.NormalizeHost(strings.TrimSpace(cmd.String("host")))
}

func newAuthStore() (*auth.Store, error) {
	configPath, err := configpath.ConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	return auth.NewStore(configPath, auth.OSSecretStore{}), nil
}

func newSonarClientFromResolvedAuth(cmd *cli.Command, resolved auth.ResolvedAuth) (*sonar.Client, error) {
	timeout, err := timeoutFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	client, clientErr := sonar.NewClient(resolved.Host, resolved.Token, timeout)
	if clientErr != nil {
		return nil, usageError(clientErr.Error(), "use --host with a full URL like https://sonarcloud.io")
	}
	return client, nil
}

func resolveAuth(cmd *cli.Command, requireOrg bool) (auth.ResolvedAuth, error) {
	store, err := newAuthStore()
	if err != nil {
		return auth.ResolvedAuth{}, runtimeError("resolve auth store", err)
	}

	resolved, err := store.Resolve(strings.TrimSpace(cmd.String("host")), strings.TrimSpace(cmd.String("org")))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrNoActiveProfile):
			return auth.ResolvedAuth{}, authError("missing Sonar token", "run `sonar-issues auth login --org <org>` or set SONAR_TOKEN")
		case errors.Is(err, auth.ErrProfileNotFound):
			return auth.ResolvedAuth{}, authError("no stored login matches the requested host and org", "run `sonar-issues auth login --org <org>` or set SONAR_TOKEN")
		default:
			return auth.ResolvedAuth{}, runtimeError("resolve authentication", err)
		}
	}

	if requireOrg && resolved.Org == "" {
		return auth.ResolvedAuth{}, usageError("missing organization", "pass --org, set SONAR_ORG, or log in with `sonar-issues auth login --org <org>`")
	}

	return resolved, nil
}

func mapSonarError(err error) error {
	var httpErr *sonar.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return authError(httpErr.Message, "check your token with `sonar-issues auth check`")
		case http.StatusNotFound:
			return notFoundError(httpErr.Message)
		}
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return runtimeError("request timed out or was canceled", err)
	}

	return runtimeError("Sonar API request failed", err)
}
