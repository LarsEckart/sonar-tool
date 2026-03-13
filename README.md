# Sonar Issues CLI

Query SonarCloud issues and projects from the command line.

This repo now contains a Go CLI with:

- secure local login via the system keychain
- environment-based auth for CI and one-off scripts
- project listing
- issue listing with common Sonar filters
- plain text, JSON, and Markdown output
- support for SonarCloud by default, plus compatible hosts via `--host`

```text
 token from env or keychain
            |
            v
      sonar-issues
       |   |   |
       |   |   +--> auth
       |   +------> projects list
       +----------> issues list
                    |
                    v
          SonarCloud / SonarQube API
```

> [!NOTE]
> This README is for the Go CLI.
> The older Node script is still in the repo as `sonar-issues.js`, but the Go tool is the current implementation.

## Install

### Build from source

This is the clearest option because it gives you the expected binary name:

```bash
go build -o sonar-issues .
./sonar-issues --help
```

### Install with Go

```bash
go install github.com/LarsEckart/sonar-tool@latest
```

> [!NOTE]
> `go install` currently produces a binary named `sonar-tool` because that is the module path.
> The CLI itself uses the command name `sonar-issues` in help and examples.
> You can either rename the binary or replace `sonar-issues` with `sonar-tool` in the examples below.

## Authentication

You can authenticate in two ways:

1. **Stored login** for local use
2. **Environment variables** for CI or one-off commands

### Option 1: stored login

Interactive login:

```bash
sonar-issues auth login --org my-org
```

Safer non-interactive login:

```bash
printf '%s' "$SONAR_TOKEN" | sonar-issues auth login --org my-org --token-stdin
```

Less safe compatibility option:

```bash
sonar-issues auth login --org my-org --with-token "$SONAR_TOKEN"
```

Useful auth commands:

```bash
sonar-issues auth current
sonar-issues auth check
sonar-issues auth check --org my-org
sonar-issues auth logout
sonar-issues auth logout --all --force
```

What gets stored:

- **token**: system keychain
- **active profile metadata**: `~/.config/sonar-issues/config.json`

The config file stores host and org metadata, not the token itself.

### Option 2: environment variables

This is handy for CI, scripts, and temporary shells:

```bash
export SONAR_TOKEN="your_token_here"
export SONAR_ORG="my-org"
export SONAR_HOST_URL="https://sonarcloud.io"   # optional
export SONAR_TIMEOUT="30"                       # optional, seconds
```

Backward-compatible token alias:

```bash
export SONAR_TOOL_TOKEN="your_token_here"
```

## Quick start

```bash
# 1) log in once
sonar-issues auth login --org my-org

# 2) verify auth
sonar-issues auth check

# 3) list projects in the active org
sonar-issues projects list

# 4) list issues for one project
sonar-issues issues list --project my-project
```

## Commands

### `auth`

Manage stored Sonar auth profiles.

```bash
sonar-issues auth login --org my-org
sonar-issues auth current
sonar-issues auth check
sonar-issues auth logout
```

### `projects list`

List projects in an organization.

Plain output prints one project key per line, which makes it easy to pipe into other tools.

```bash
# use org from active login or SONAR_ORG
sonar-issues projects list

# explicit org
sonar-issues projects list --org my-org

# paginate
sonar-issues projects list --org my-org --limit 50 --page 2

# fetch all pages
sonar-issues projects list --org my-org --all

# JSON for scripts
sonar-issues projects list --org my-org --json

# Markdown for docs or tickets
sonar-issues projects list --org my-org --markdown
```

### `issues list`

List issues for a project or across an organization.

Common cases:

```bash
# issues for one project
sonar-issues issues list --project my-project

# full plain-text details
sonar-issues issues list --project my-project --full

# org-wide search
sonar-issues issues list --org my-org

# JSON for scripts
sonar-issues issues list --project my-project --json

# Markdown report
sonar-issues issues list --project my-project --markdown > issues.md

# fetch all pages
sonar-issues issues list --project my-project --all --json
```

## Common issue filters

### Scope

```bash
sonar-issues issues list --project my-project
sonar-issues issues list --project my-project --branch feature/my-branch
sonar-issues issues list --project my-project --pr 123
```

### Severity and type

```bash
sonar-issues issues list --project my-project --types BUG,VULNERABILITY
sonar-issues issues list --project my-project --severities CRITICAL,BLOCKER
sonar-issues issues list --project my-project --impact-severities HIGH,BLOCKER
sonar-issues issues list --project my-project --qualities SECURITY
```

### Status and ownership

```bash
sonar-issues issues list --project my-project --unresolved
sonar-issues issues list --project my-project --resolved=true
sonar-issues issues list --project my-project --statuses OPEN,CONFIRMED
sonar-issues issues list --project my-project --assignee __me__
sonar-issues issues list --project my-project --author dev@example.com
```

### Rules, tags, and languages

```bash
sonar-issues issues list --project my-project --rules java:S1068
sonar-issues issues list --project my-project --tags security,owasp
sonar-issues issues list --project my-project --languages go,ts
```

### Time filters

```bash
sonar-issues issues list --project my-project --new
sonar-issues issues list --project my-project --created-after 2026-01-01
sonar-issues issues list --project my-project --created-before 2026-01-31
sonar-issues issues list --project my-project --created-in-last 7d
sonar-issues issues list --project my-project --created-in-last 1m2w
```

### Sorting and paging

```bash
sonar-issues issues list --project my-project --sort CREATION_DATE --desc
sonar-issues issues list --project my-project --limit 20 --page 2
sonar-issues issues list --project my-project --all
```

## Output modes

The tool supports three output modes:

- **plain text**: default, easy to scan
- **JSON**: use `--json` for scripts and agents
- **Markdown**: use `--markdown` for reports, tickets, and docs

`--json` and `--markdown` are mutually exclusive.

### Plain text

Good for terminal use.

```bash
sonar-issues issues list --project my-project
sonar-issues issues list --project my-project --full
```

### JSON

Good for `jq`, scripts, and automation.

```bash
sonar-issues issues list --project my-project --json | jq '.issues[] | .key'
sonar-issues projects list --org my-org --json | jq '.projects[] | .key'
```

### Markdown

Good for GitHub, Jira, and docs.

```bash
sonar-issues issues list --project my-project --markdown > sonar-report.md
```

## Global flags

These flags work across commands:

| Flag | Meaning |
| --- | --- |
| `--host` | Base URL, defaults to `https://sonarcloud.io` |
| `--org` | Default organization key |
| `--timeout` | HTTP timeout in seconds |
| `--json` | Structured JSON output |
| `--markdown` / `--md` | Markdown output |
| `--quiet` / `-q` | Suppress non-data success output |
| `--verbose` / `-v` | More diagnostics to stderr |
| `--version` | Print the version |

## Environment variables

| Variable | Use |
| --- | --- |
| `SONAR_TOKEN` | Token for the current process |
| `SONAR_TOOL_TOKEN` | Backward-compatible token alias |
| `SONAR_ORG` | Default org |
| `SONAR_HOST_URL` | Default host |
| `SONAR_TIMEOUT` | Default timeout in seconds |
| `XDG_CONFIG_HOME` | Changes where config metadata is stored |

## Useful patterns

List projects, then inspect one project:

```bash
sonar-issues projects list --org my-org
sonar-issues issues list --org my-org --project my-project
```

Generate a Markdown report for a PR:

```bash
sonar-issues issues list \
  --project my-project \
  --pr 123 \
  --markdown > pr-sonar-report.md
```

Find unresolved critical bugs:

```bash
sonar-issues issues list \
  --project my-project \
  --types BUG \
  --severities CRITICAL,BLOCKER \
  --unresolved
```

## Current behavior notes

> [!NOTE]
> A stored active profile can supply both host and org defaults.
> You can still override either one with `--host` or `--org`.

> [!NOTE]
> The current text output is already plain and uncolored.
> `--no-color` is accepted as a global flag, but it does not change visible output today.

## Exit codes

| Code | Meaning |
| ---: | --- |
| `0` | Success |
| `1` | Runtime or API failure |
| `2` | Invalid usage |
| `3` | Auth failure |
| `4` | Not found or no access |
| `130` | Interrupted |

## Development

Run tests:

```bash
go test ./...
```

Helpful top-level files and directories:

- `cmd/` - CLI commands and flag wiring
- `internal/` - auth, Sonar client, formatters, validation
- `tests/` - subprocess CLI tests and fixtures
- `CLI_SPEC.md` - planned CLI shape and design notes
- `sonar-issues.js` - legacy Node script
