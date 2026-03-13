# Sonar Issues CLI Spec

Status: draft v1

This document defines the Go rewrite of `sonar-issues.js`.

## 1. Goal

Build a small, machine-friendly Go CLI for querying SonarCloud issues and projects.

Primary use cases:
- fetch issues for one project
- fetch issues across an organization
- list projects in an organization
- filter aggressively for scripting and reporting
- produce stable JSON for agents/scripts
- still have readable plain text and markdown for humans

## 2. Design choices taken from the existing tools

From `hccli`:
- use `urfave/cli/v3`
- thin `cmd` layer, API logic in a separate package
- global flags for host and timeout
- JSON as a first-class output mode
- subprocess CLI tests against the built binary
- large JSON output warning + temp file spillover

From `jirafluence`:
- validated domain types at the boundary
- explicit parse/normalize helpers for flags and URLs
- human-friendly error messages instead of raw transport noise
- keychain-backed auth with a small injectable store layer
- focused unit tests for parsing and formatting
- small readable packages, not a giant `main.go`

## 3. Non-goals

- no interactive TUI
- no issue mutation commands in v1
- no attempt to be a general Sonar admin tool
- no public Go SDK commitment; internal packages are fine

## 4. Proposed binary name

`sonar-issues`

Reason: keep continuity with the current script and existing muscle memory.

## 5. UX principles

- humans first, but scripts are a first-class use case
- stdout carries primary data
- stderr carries warnings, hints, and errors
- avoid secrets via flags; the only compatibility exception is `auth login --with-token`, and it should be documented as less safe than prompt/stdin
- stable machine mode via `--json`
- explicit subcommands instead of overloaded flags
- fail early on invalid filter values

## 6. Command tree

```text
sonar-issues [global flags] <command> [subcommand] [flags]

Commands:
  issues list        List Sonar issues
  projects list      List Sonar projects
  auth login         Save token to system keychain and select active profile
  auth logout        Remove stored auth for a profile
  auth current       Show the active auth profile (never the token)
  auth check         Validate auth and optionally org access
  help               Show help for a command
```

### Why this shape

The current JS script overloads `-p` so that `-p` alone means “list projects”.
That is convenient but not a good long-term CLI contract.

Use explicit subcommands instead:
- `projects list` for projects
- `issues list` for issues

This is clearer, easier to document, and easier to extend.

## 7. Global flags

Available on all commands.

| Flag | Type | Default | Notes |
|---|---:|---|---|
| `--host` | string | active profile host, else `https://sonarcloud.io` | Base URL for SonarCloud or compatible SonarQube host |
| `--org` | string | flag > env > active profile org | Default org for commands that operate on an org |
| `--timeout` | duration seconds | `30` | HTTP timeout in seconds |
| `--json` | bool | `false` | Structured JSON output |
| `--markdown` | bool | `false` | Human report output |
| `--no-color` | bool | `false` | Disable color in human output |
| `-q, --quiet` | bool | `false` | Suppress non-data success chatter |
| `-v, --verbose` | bool | `false` | More diagnostics to stderr |
| `-h, --help` | bool | | Standard help |
| `--version` | bool | | Print version |

### Output mode rule

`--json` and `--markdown` are mutually exclusive.

If neither is set, output is plain text.

## 8. Environment variables

| Variable | Purpose |
|---|---|
| `SONAR_TOKEN` | Preferred per-process token override |
| `SONAR_TOOL_TOKEN` | Backward-compatible alias from current script |
| `SONAR_ORG` | Default organization |
| `SONAR_HOST_URL` | Default host if `--host` is omitted |
| `SONAR_TIMEOUT` | Default timeout in seconds |
| `NO_COLOR` | Disable color |

### Auth precedence for normal commands

Highest to lowest:
1. `SONAR_TOKEN`
2. `SONAR_TOOL_TOKEN`
3. active stored login profile

If no token is available, fail with an actionable message.

Do not support a global `--token` flag.

## 8.1 Auth storage model

Use the same overall model as the official Sonar CLI:
- `auth login` persists credentials
- the token is stored in the system keychain
- later commands reuse that stored login automatically

### Storage details

**Secret storage:** system keychain via `github.com/zalando/go-keyring`
- macOS: Keychain
- Linux: Secret Service
- Windows: Credential Manager

**Non-secret metadata:** XDG-style config file
- path: `~/.config/sonar-issues/config.json`
- stores active profile metadata only
- never stores the token itself

Suggested config shape:

```json
{
  "active_profile": "sonarcloud:example-org",
  "profiles": {
    "sonarcloud:example-org": {
      "host": "https://sonarcloud.io",
      "org": "example-org"
    }
  }
}
```

Suggested keychain naming:
- service: `sonar-issues`
- key/account: `token:<host>:<org>`

This gives us:
- official-CLI-style login UX
- secure local storage
- support for multiple org profiles later without redesign
- env vars still available for CI and one-off overrides

## 9. Command specs

## 9.1 `issues list`

List issues for a project or an organization.

### Usage

```text
sonar-issues issues list [flags]
```

### Required inputs

At least one of:
- `--project <key>`
- `--org <key>`

If both are set, search is scoped to the project within the org.

### Flags

| Flag | Type | Default | Notes |
|---|---:|---|---|
| `--project` | string | unset | Sonar project key |
| `--branch` | string | unset | Branch name |
| `--pull-request`, `--pr` | string | unset | Pull request ID |
| `--types` | csv | unset | `CODE_SMELL,BUG,VULNERABILITY,SECURITY_HOTSPOT` |
| `--severities` | csv | unset | Legacy severity values |
| `--impact-severities` | csv | unset | `INFO,LOW,MEDIUM,HIGH,BLOCKER` |
| `--qualities` | csv | unset | `MAINTAINABILITY,RELIABILITY,SECURITY` |
| `--statuses` | csv | unset | `OPEN,CONFIRMED,FALSE_POSITIVE,ACCEPTED,FIXED` |
| `--tags` | csv | unset | Tag filter |
| `--rules` | csv | unset | Rule keys like `java:S1068` |
| `--assignee` | string | unset | Assignee login; `__me__` allowed |
| `--author` | string | unset | SCM author |
| `--languages` | csv | unset | Language keys like `java,js,py` |
| `--created-after` | date | unset | `YYYY-MM-DD` |
| `--created-before` | date | unset | `YYYY-MM-DD` |
| `--created-in-last` | duration span | unset | e.g. `7d`, `2w`, `1m2w` |
| `--new` | bool | `false` | Alias for Sonar `sinceLeakPeriod=true` |
| `--resolved` | bool | unset | Explicit resolved filter |
| `--unresolved` | bool | `false` | Shortcut for `--resolved=false` |
| `--sort` | string | unset | Sonar sort field |
| `--asc` | bool | `false` | Ascending sort |
| `--desc` | bool | `false` | Descending sort |
| `--limit` | int | `100` | Per page; cap at `500` |
| `--page` | int | `1` | 1-based page index |
| `--all` | bool | `false` | Fetch all pages until exhausted |
| `--full` | bool | `false` | Plain/markdown only; include more issue detail |

### Validation rules

- `--asc` and `--desc` cannot both be set
- `--resolved` and `--unresolved` cannot conflict
- `--branch` and `--pull-request` may both be passed through if Sonar supports it; otherwise reject early if API docs disallow the combo
- `--limit` must be `1..500`
- `--page` must be `>= 1`
- CSV values are normalized to uppercase where appropriate
- enum-like values are validated before making the request

### Semantics

- default mode fetches exactly one Sonar page
- `--all` repeatedly fetches pages and aggregates results
- JSON mode returns a stable envelope, even when `--all` is used
- plain/markdown mode should indicate total count and whether output is truncated to one page

### JSON contract

`--json` prints a stable structure:

```json
{
  "query": {
    "org": "my-org",
    "project": "my-project",
    "page": 1,
    "limit": 100,
    "all": false
  },
  "paging": {
    "page_index": 1,
    "page_size": 100,
    "total": 123
  },
  "issues": [],
  "rules": []
}
```

Notes:
- field names should be snake_case in the CLI envelope for long-term stability
- raw Sonar fields inside issue objects may remain API-native if that reduces translation work
- if needed, include a top-level `next_page` integer or `null`

### Plain output

Default plain output is concise and scan-friendly:

```text
Found 123 issues (showing 100 on page 1)

1. MAJOR CODE_SMELL  src/foo/bar.go:42
   Rule: go:S1234
   Message: remove this unused field

2. CRITICAL BUG  src/main.go:18
   Rule: go:S5678
   Message: possible nil dereference

... 23 more issues. Use --page 2 or --all.
```

With `--full`, include:
- rule key + rule name
- severity / impact summary
- type
- status
- file + line
- effort
- tags
- assignee / author when present
- created / updated
- issue key

### Markdown output

Markdown is for reports and copy/paste into tickets/docs.

Behavior:
- include a summary section
- include one section per issue
- be readable when pasted into GitHub, Jira, or Confluence
- no ANSI color in markdown mode

## 9.2 `projects list`

List projects in an organization.

### Usage

```text
sonar-issues projects list [flags]
```

### Required inputs

- `--org <key>` unless provided by env/config

### Flags

| Flag | Type | Default | Notes |
|---|---:|---|---|
| `--org` | string | `SONAR_ORG` | Organization key |
| `--limit` | int | `100` | Per page; cap at `500` |
| `--page` | int | `1` | 1-based page index |
| `--all` | bool | `false` | Fetch all pages |

### JSON contract

```json
{
  "query": {
    "org": "my-org",
    "page": 1,
    "limit": 100,
    "all": false
  },
  "paging": {
    "page_index": 1,
    "page_size": 100,
    "total": 250
  },
  "projects": [
    {
      "key": "proj-a",
      "name": "Project A"
    }
  ]
}
```

### Plain output

Default plain output prints one project key per line.

This makes piping simple:

```text
proj-a
proj-b
proj-c
# ... 147 more. Use --page 2 or --all.
```

### Markdown output

```markdown
## Projects

- `proj-a`
- `proj-b`
- `proj-c`
```

## 9.3 `auth login`

Save a token to the system keychain and make that profile active.

### Usage

```text
sonar-issues auth login --org <org> [flags]
```

### Flags

| Flag | Type | Default | Notes |
|---|---:|---|---|
| `--org`, `-o` | string | required for SonarCloud | Organization key |
| `--host`, `-s` | string | `https://sonarcloud.io` | Host/profile scope |
| `--with-token`, `-t` | string | unset | Compatibility with official CLI; less safe because it may leak via shell history/process list |
| `--token-stdin` | bool | `false` | Read token from stdin |

### Behavior

- if `--with-token` is set, use it directly
- else if `--token-stdin` is set, read the full token from stdin and trim trailing newline
- else if stdin is a TTY, prompt securely without echo
- store token in system keychain
- update config so this host+org becomes the active profile
- overwrite existing token for the same host+org

### Notes

For safety, help text should prefer:
1. secure prompt
2. `--token-stdin`
3. `--with-token` as a compatibility escape hatch

## 9.4 `auth current`

Show the currently active auth profile.

### Usage

```text
sonar-issues auth current
```

### Behavior

- prints active host and org
- never prints the token
- if no active profile exists, exits non-zero with a helpful message

## 9.5 `auth logout`

Remove stored auth for one profile or all profiles.

### Usage

```text
sonar-issues auth logout [flags]
```

### Flags

| Flag | Type | Default | Notes |
|---|---:|---|---|
| `--org`, `-o` | string | active profile org | Target org |
| `--host`, `-s` | string | active profile host | Target host |
| `--all` | bool | `false` | Remove all stored profiles |
| `--force`, `-f` | bool | `false` | Skip confirmation |

### Behavior

- remove token(s) from keychain
- remove profile metadata from config
- if the active profile is removed, clear or switch active profile deterministically
- if interactive and `--all` is used without `--force`, confirm first

## 9.6 `auth check`

Validate credentials and optionally verify org access.

### Usage

```text
sonar-issues auth check [flags]
```

### Flags

| Flag | Type | Default | Notes |
|---|---:|---|---|
| `--org` | string | flag > env > active profile org | If set, verify this org is accessible |

### Behavior

- resolves auth using env override or stored active profile
- checks that a token can be loaded
- calls a cheap Sonar endpoint
- if `--org` is provided, verifies the org exists / is visible
- plain output should be one short success line
- JSON output should return a small status object

## 10. Help text expectations

Every command help should contain:
- one sentence purpose
- 4–8 realistic examples
- the most important flags first
- clear env var notes for auth-sensitive commands

If a required arg is missing:
- print a short error
- print one relevant example
- end with `Use --help for more information.`

## 11. Error handling

Expected errors should be rewritten for humans.

Examples:
- bad token source
- invalid org/project key
- invalid enum values in filters
- timeout
- Sonar API 401/403/404
- invalid JSON from server

### Error message style

```text
error: missing Sonar token
hint: run `sonar-issues auth login --org <org>` or set SONAR_TOKEN
```

```text
error: invalid value for --types: "BUGS"
allowed: BUG, CODE_SMELL, SECURITY_HOTSPOT, VULNERABILITY
```

Do not dump stack traces by default.

## 12. Exit codes

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | Runtime/API failure |
| `2` | Invalid usage / validation error |
| `3` | Auth failure |
| `4` | Not found / no access |
| `130` | Interrupted by Ctrl-C |

Keep the map small and stable.

## 13. Output rules

### stdout

- primary command output only
- JSON goes to stdout
- plain project list goes to stdout, one item per line

### stderr

- validation errors
- auth errors
- pagination / truncation hints
- large output warnings
- verbose diagnostics

## 14. Large output behavior

Adopt the `hccli` idea.

When JSON output exceeds ~30KB:
- still print the full JSON to stdout
- also write it to a temp file
- print a warning to stderr with the temp file path
- print a short hint suggesting `--limit`, `--page`, or tighter filters

This helps when running inside agent shells with output truncation.

## 15. Suggested package layout

```text
.
├── main.go
├── cmd/
│   ├── root.go
│   ├── issues.go
│   ├── projects.go
│   ├── auth.go
│   ├── shared.go
│   └── json.go
├── internal/
│   ├── sonar/
│   │   ├── client.go
│   │   ├── issues.go
│   │   ├── projects.go
│   │   └── auth.go
│   ├── auth/
│   │   ├── keyring.go
│   │   ├── store.go
│   │   ├── config.go
│   │   └── resolver.go
│   ├── domain/
│   │   ├── keys.go
│   │   ├── query.go
│   │   ├── enums.go
│   │   └── validation.go
│   ├── format/
│   │   ├── issues_plain.go
│   │   ├── issues_markdown.go
│   │   └── projects_plain.go
│   └── config/
│       └── path.go
└── tests/
    ├── cli_test.go
    ├── issues_cli_test.go
    ├── projects_cli_test.go
    ├── auth_cli_test.go
    └── large_output_cli_test.go
```

### Architecture intent

```text
urfave CLI commands
    -> parse flags
    -> validate/normalize into domain query
    -> call internal/sonar client
    -> send result to formatter/output layer
```

### Rules

- `cmd/` should not construct raw URLs by hand
- `internal/sonar/` owns Sonar endpoint paths and transport details
- `internal/auth/` owns keychain persistence, active profile config, and auth resolution
- `internal/domain/` owns validation and normalized query types
- `internal/format/` owns plain + markdown rendering
- API methods accept `context.Context` first
- HTTP/config dependencies are injected where easy, but avoid overengineering

## 16. Data model guidelines

Prefer explicit types over loose strings at the boundary.

Examples:
- `type OrganizationKey string`
- `type ProjectKey string`
- `type OutputMode string`
- `type SortField string`
- `type IssueType string`
- `type Severity string`

Use parse helpers such as:
- `ParseProjectKey(string) (ProjectKey, error)`
- `ParseCSVEnum[T ...](string) ([]T, error)`
- `NormalizeIssuesQuery(...) (domain.IssuesQuery, error)`

Inside the transport layer, preserve raw Sonar field names where useful.

## 17. HTTP client rules

- one shared client type in `internal/sonar/client.go`
- configurable base URL
- configurable timeout
- bearer auth header
- decode non-2xx responses into helpful errors
- wrap all transport errors with context
- include request URL path in verbose logs only, not default output

Potential helper shape:

```text
Client.do(ctx, method, path, query, out)
```

Keep it simple; no premature generics needed unless they clearly reduce duplication.

## 18. Testing strategy

Adopt `hccli` here.

### CLI tests

- build the binary once in `tests/cli_test.go`
- run it as a subprocess
- assert on stdout, stderr, and exit code
- cover:
  - missing token with no stored profile
  - invalid flags
  - auth login / current / logout
  - auth check
  - issues list JSON
  - projects list plain output
  - timeout handling
  - large output temp file behavior

### Unit tests

Adopt the `jirafluence` discipline here.

Test small pure helpers directly:
- enum parsing
- query normalization
- date/span parsing
- output formatting
- pagination merging

## 19. Recommended v1 implementation order

1. root app + global config loading
2. `auth login` / `auth current` / `auth logout`
3. `auth check`
4. `projects list`
5. `issues list` plain mode
6. `issues list --json`
7. `issues list --markdown`
8. `--all` pagination
9. large output spillover
10. full test suite + README update

## 20. Example invocations

```bash
# login like the official sonar CLI
sonar-issues auth login -o example-org --with-token "$SONAR_TOKEN"

# safer non-interactive login
printf '%s' "$SONAR_TOKEN" | sonar-issues auth login -o example-org --token-stdin

# show active profile
sonar-issues auth current

# verify token works
sonar-issues auth check

# verify org access
sonar-issues auth check --org my-org

# list projects in org
sonar-issues projects list --org my-org

# list all projects in org as JSON
sonar-issues projects list --org my-org --all --json

# list issues for one project
sonar-issues issues list --project my-project --org my-org

# list only unresolved critical bugs
sonar-issues issues list \
  --project my-project \
  --types BUG \
  --severities CRITICAL,BLOCKER \
  --unresolved

# issues on new code only
sonar-issues issues list --project my-project --new

# issues for a pull request
sonar-issues issues list --project my-project --pr 123 --json

# markdown report
sonar-issues issues list --project my-project --full --markdown > report.md

# fetch everything for scripting
sonar-issues issues list --org my-org --all --json | jq '.issues[] | .key'
```

## 21. Open questions for later, not blockers

- whether to support SonarQube-specific differences beyond a custom `--host`
- whether `SECURITY_HOTSPOT` belongs in the same command or a later dedicated command
- whether markdown output should include issue URLs if Sonar provides enough data
- whether `--format json|plain|markdown` is worth standardizing in v2 instead of separate booleans

## 22. Summary

Build a small CLI with this shape:

```text
machine-friendly contract from hccli
+ clear domain validation from jirafluence
+ explicit subcommands instead of overloaded flags
```

That should give a solid v1 without painting the tool into a corner.
