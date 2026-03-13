package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lars/sonar-tool/internal/auth"
	"github.com/lars/sonar-tool/internal/domain"
	cli "github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func newAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Manage Sonar authentication profiles",
		Commands: []*cli.Command{
			{
				Name:        "login",
				Usage:       "Save a token to the system keychain and make the profile active",
				Description: "Stores a Sonar token in the system keychain and records the active host/org profile in ~/.config/sonar-issues/config.json.",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "with-token", Aliases: []string{"t"}, Usage: "Token value passed directly on the command line (less safe)"},
					&cli.BoolFlag{Name: "token-stdin", Usage: "Read the token from stdin"},
				},
				Action: authLoginAction,
			},
			{
				Name:   "current",
				Usage:  "Show the active auth profile",
				Action: authCurrentAction,
			},
			{
				Name:        "logout",
				Usage:       "Remove stored auth for one profile or all profiles",
				Description: "Deletes tokens from the system keychain and removes profile metadata from local config.",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "all", Usage: "Remove all stored profiles"},
					&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Skip confirmation prompts"},
				},
				Action: authLogoutAction,
			},
			{
				Name:        "check",
				Usage:       "Validate credentials and optionally verify org access",
				Description: "Uses env token overrides first, then stored profiles, and checks that the token works against Sonar.",
				Action:      authCheckAction,
			},
		},
	}
}

func authLoginAction(_ context.Context, cmd *cli.Command) error {
	store, err := newAuthStore()
	if err != nil {
		return runtimeError("resolve auth store", err)
	}

	host, err := domain.NormalizeHost(cmd.String("host"))
	if err != nil {
		return usageError(err.Error(), "use --host with a full URL like https://sonarcloud.io")
	}

	org, err := domain.ParseOrganizationKey(cmd.String("org"))
	if err != nil {
		return usageError(err.Error(), "pass --org <org>")
	}

	token, err := loginToken(cmd)
	if err != nil {
		return err
	}

	if err := store.Login(host, string(org), token); err != nil {
		return runtimeError("save login", err)
	}

	payload := map[string]any{
		"ok":   true,
		"host": host,
		"org":  string(org),
		"active_profile": map[string]string{
			"host": host,
			"org":  string(org),
		},
	}
	return writeOutput(cmd, payload, fmt.Sprintf("Saved login for %s on %s", org, host))
}

func authCurrentAction(_ context.Context, cmd *cli.Command) error {
	store, err := newAuthStore()
	if err != nil {
		return runtimeError("resolve auth store", err)
	}

	profile, err := store.CurrentProfile()
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveProfile) {
			return authError("no active auth profile", "run `sonar-issues auth login --org <org>`")
		}
		return runtimeError("load active profile", err)
	}

	payload := map[string]any{
		"active_profile": profile,
	}
	plain := fmt.Sprintf("Active profile\nhost: %s\norg:  %s", profile.Host, profile.Org)
	return writeOutput(cmd, payload, plain)
}

func authLogoutAction(_ context.Context, cmd *cli.Command) error {
	store, err := newAuthStore()
	if err != nil {
		return runtimeError("resolve auth store", err)
	}

	if cmd.Bool("all") {
		if err := confirmLogoutAll(cmd); err != nil {
			return err
		}
		if err := store.LogoutAll(); err != nil {
			return runtimeError("remove stored auth", err)
		}
		return writeOutput(cmd, map[string]any{"ok": true, "removed": "all"}, "Removed all stored auth profiles")
	}

	target, err := store.ResolveTargetProfile(cmd.String("host"), cmd.String("org"))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrNoActiveProfile):
			return authError("no active auth profile", "pass --host and --org, or log in first")
		case errors.Is(err, auth.ErrProfileNotFound):
			return notFoundError("the requested auth profile was not found")
		default:
			return runtimeError("resolve target profile", err)
		}
	}

	if err := store.Logout(target.Host, target.Org); err != nil {
		if errors.Is(err, auth.ErrProfileNotFound) {
			return notFoundError("the requested auth profile was not found")
		}
		return runtimeError("remove stored auth", err)
	}

	payload := map[string]any{
		"ok":      true,
		"removed": map[string]string{"host": target.Host, "org": target.Org},
	}
	plain := fmt.Sprintf("Removed stored auth for %s on %s", target.Org, target.Host)
	return writeOutput(cmd, payload, plain)
}

func authCheckAction(ctx context.Context, cmd *cli.Command) error {
	resolved, err := resolveAuth(cmd, false)
	if err != nil {
		return err
	}

	client, err := newSonarClientFromResolvedAuth(cmd, resolved)
	if err != nil {
		return err
	}

	if err := client.ValidateAuth(ctx); err != nil {
		return mapSonarError(err)
	}
	if resolved.Org != "" {
		if err := client.CheckOrganizationAccess(ctx, resolved.Org); err != nil {
			return mapSonarError(err)
		}
	}

	payload := map[string]any{
		"ok":           true,
		"host":         resolved.Host,
		"org":          resolved.Org,
		"token_source": resolved.TokenSource,
		"org_checked":  resolved.Org != "",
	}
	plain := fmt.Sprintf("Authentication OK for %s", resolved.Host)
	if resolved.Org != "" {
		plain = fmt.Sprintf("Authentication OK for %s (org: %s)", resolved.Host, resolved.Org)
	}
	return writeOutput(cmd, payload, plain)
}

func loginToken(cmd *cli.Command) (string, error) {
	withToken := strings.TrimSpace(cmd.String("with-token"))
	fromStdin := cmd.Bool("token-stdin")
	if withToken != "" && fromStdin {
		return "", usageError("--with-token and --token-stdin cannot be used together", "use the prompt, stdin, or one explicit token source")
	}
	if withToken != "" {
		return withToken, nil
	}
	if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", runtimeError("read token from stdin", err)
		}
		token := strings.TrimSpace(string(data))
		if token == "" {
			return "", usageError("token from stdin was empty", "pipe a token into `sonar-issues auth login --token-stdin`")
		}
		return token, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", authError("missing Sonar token", "use `--token-stdin`, `--with-token`, or run the command in a terminal for a secure prompt")
	}

	_, _ = fmt.Fprint(cmd.Root().ErrWriter, "Sonar token: ")
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(cmd.Root().ErrWriter)
	if err != nil {
		return "", runtimeError("read token from terminal", err)
	}

	token := strings.TrimSpace(string(bytes))
	if token == "" {
		return "", usageError("token cannot be empty", "paste a valid Sonar token")
	}
	return token, nil
}

func confirmLogoutAll(cmd *cli.Command) error {
	if cmd.Bool("force") || !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}

	_, _ = fmt.Fprint(cmd.Root().ErrWriter, "Remove all stored auth profiles? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return runtimeError("read confirmation", err)
	}

	normalized := strings.ToLower(strings.TrimSpace(answer))
	if normalized != "y" && normalized != "yes" {
		return interruptedError()
	}
	return nil
}

func writeOutput(cmd *cli.Command, payload any, plain string) error {
	mode, err := outputMode(cmd)
	if err != nil {
		return err
	}

	switch mode {
	case OutputModeJSON:
		encoder := json.NewEncoder(cmd.Root().Writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(payload)
	default:
		if cmd.Bool("quiet") {
			return nil
		}
		_, err := fmt.Fprintln(cmd.Root().Writer, plain)
		return err
	}
}
