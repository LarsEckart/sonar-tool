# AGENTS.md

Developer notes for the Go CLI in this repo.

## What this project is

This repo contains a Go rewrite of the older `sonar-issues.js` script.

Current goal:
- query Sonar issues
- query Sonar projects
- support secure local auth
- support script-friendly JSON output
- keep the CLI small and readable

Default host is SonarCloud, but the CLI also accepts a custom `--host` for compatible SonarQube-style endpoints.

## Current command surface

Implemented commands:

```text
sonar-issues auth login
sonar-issues auth current
sonar-issues auth logout
sonar-issues auth check
sonar-issues projects list
sonar-issues issues list
```

Global flags are defined in `cmd/root.go`.

## High-level architecture

```text
main.go
  -> cmd/
      -> parse flags
      -> resolve auth / output mode / timeout
      -> call internal packages

internal/domain/
  -> normalize and validate user input

internal/auth/
  -> config file + keychain access + auth resolution

internal/sonar/
  -> HTTP client + Sonar endpoint calls

internal/format/
  -> plain text and markdown rendering
```

Keep `cmd/` thin.
Do validation in `internal/domain/`.
Do transport and API mapping in `internal/sonar/`.
Do not spread Sonar URL/query construction across command files.

## Important files

- `main.go` - app entrypoint and exit handling
- `cmd/root.go` - root command and global flags
- `cmd/shared.go` - shared CLI helpers, error mapping, auth/client setup
- `cmd/auth.go` - auth subcommands
- `cmd/projects.go` - project listing command
- `cmd/issues.go` - issue listing command
- `internal/auth/` - keychain + config storage
- `internal/domain/issues.go` - issue query normalization and validation
- `internal/sonar/` - HTTP client and Sonar API calls
- `internal/format/` - plain/markdown output
- `tests/` - subprocess CLI tests with fixtures
- `CLI_SPEC.md` - design target, not always identical to current behavior

## Auth model

Tokens are stored in the system keychain.
Non-secret profile metadata is stored in:

```text
~/.config/sonar-issues/config.json
```

The config file must never store the token.

### Actual auth resolution behavior

Read the code before changing docs here. Current behavior is:

1. host: `--host` -> `SONAR_HOST_URL` -> active profile -> default host
2. org: `--org` -> `SONAR_ORG` -> active profile
3. if the resolved host+org matches a stored profile, use the keychain token
4. else try `SONAR_TOKEN`
5. else try `SONAR_TOOL_TOKEN`

This means the current implementation prefers a matching stored profile over env tokens.
That is different from parts of `CLI_SPEC.md`, so do not assume the spec is the source of truth.

## Output modes

Supported output modes:
- plain text
- JSON via `--json`
- Markdown via `--markdown` / `--md`

`--json` and `--markdown` are mutually exclusive.

Plain and Markdown rendering live in `internal/format/`.
JSON envelopes are built in `cmd/issues.go` and `cmd/projects.go`.

## Testing

Run the full test suite with:

```bash
go test ./...
```

Test layout:
- unit tests for auth/domain/sonar helpers in `internal/...`
- subprocess CLI tests in `tests/`
- fixture-backed HTTP server in `tests/helpers_test.go`

### Refreshing sanitized live fixtures

There is a recorder test:

```bash
SONAR_RECORD_FIXTURES=1 go test ./tests -run TestRecordLiveFixtures -count=1
```

Optional env:

```bash
SONAR_FIXTURE_PROJECT=your-project-key
SONAR_TIMEOUT=30
```

This uses your locally resolved auth, fetches live data, and writes sanitized fixtures to `tests/testdata/`.

## Developer workflow

When you add or change CLI behavior:

1. update validation in `internal/domain/` if needed
2. update Sonar client code in `internal/sonar/` if needed
3. keep `cmd/` focused on wiring, not business logic
4. update tests
5. update `README.md`
6. check whether `CLI_SPEC.md` still matches reality

## Known gotchas

### Binary name mismatch

The CLI name is `sonar-issues`, but:

```bash
go install github.com/LarsEckart/sonar-tool@latest
```

currently installs a binary named `sonar-tool` because that is the module path.

If you change naming, update:
- `README.md`
- help text examples
- any install instructions

### Spec vs implementation

`CLI_SPEC.md` is useful, but it describes more than the code currently implements.
Examples of spec items that are not fully implemented yet include:
- richer help text with many examples
- large JSON spillover to temp files
- some planned package/file splits from the spec

Prefer the code and tests over the spec when documenting current behavior.

### `--no-color`

The flag exists, but current human output is plain text already.
If you add colored output later, make sure `--no-color` and `NO_COLOR` work consistently.

## Style rules for this repo

- prefer small, readable functions
- prefer explicit types and validation at boundaries
- keep error messages human-friendly
- return stable JSON envelopes for script-facing commands
- avoid clever abstractions unless they remove real duplication
- preserve secure handling of tokens

## If you add new commands

Follow the existing shape:

```text
cmd/<command>.go
  -> parse flags
  -> normalize/validate input
  -> resolve auth
  -> call sonar client
  -> format output
```

Also add subprocess coverage in `tests/` for:
- success path
- invalid usage
- auth failure when relevant
- JSON output contract when relevant
