# SonarCloud Issues Tool

Simple CLI tool to retrieve issues from SonarCloud. No dependencies required (uses Node.js built-ins).

## Setup

1. Generate a token at https://sonarcloud.io/account/security

2. Set environment variables:
```bash
export SONAR_TOKEN="your_token_here"
export SONAR_ORG="your_organization"  # optional default
```

3. Add to PATH (optional):
```bash
export PATH="$PATH:/path/to/sonar-tool"
```

## Usage

```bash
# Basic: get issues for a project
sonar-issues.js -p my_project -o my-org

# Filter by type (BUG, VULNERABILITY, CODE_SMELL)
sonar-issues.js -p my_project --types BUG,VULNERABILITY

# Filter by severity
sonar-issues.js -p my_project --severities CRITICAL,BLOCKER

# Only unresolved issues
sonar-issues.js -p my_project --unresolved

# New issues since leak period (new code)
sonar-issues.js -p my_project --new

# Issues from a specific branch
sonar-issues.js -p my_project --branch feature/my-branch

# Issues from a pull request
sonar-issues.js -p my_project --pr 123

# Filter by software quality impact
sonar-issues.js -p my_project --qualities SECURITY

# Issues created in the last week
sonar-issues.js -p my_project --created-in-last 7d

# Limit results
sonar-issues.js -p my_project -n 20

# JSON output (for scripting)
sonar-issues.js -p my_project --json

# Markdown output (for reports)
sonar-issues.js -p my_project --markdown > report.md

# All issues in an organization
sonar-issues.js -o my-org

# List all projects in an organization
sonar-issues.js -o my-org -p

# Full details on issues
sonar-issues.js -p my_project --full
```

## Output Format

Default (concise):
```
--- Issue 1 ---
Rule: Unused "private" fields should be removed
Message: Remove this unused "logger" private field.
File: src/main/java/com/example/MyClass.java:15
```

With `--full`:
```
--- Issue 1 ---
Rule: java:S1068
Rule Name: Unused "private" fields should be removed
Message: Remove this unused "logger" private field.
Severity: MAJOR
Impacts: MAINTAINABILITY:MEDIUM
Type: CODE_SMELL
Status: OPEN
Clean Code: INTENTIONAL / CLEAR
File: src/main/java/com/example/MyClass.java:15
Effort: 5min
Tags: unused
Created: 2024-01-15T10:30:00+0000
Key: AYx...
```

## All Options

```
Required:
  -p, --project <key>       Project key (or -p alone to list projects)
  -o, --org <key>           Organization key

Filters:
  -b, --branch <name>       Branch name
  --pr, --pull-request <id> Pull request ID
  -t, --types <list>        CODE_SMELL, BUG, VULNERABILITY
  -s, --severities <list>   INFO, MINOR, MAJOR, CRITICAL, BLOCKER
  --impact-severities <list> INFO, LOW, MEDIUM, HIGH, BLOCKER
  --qualities <list>        MAINTAINABILITY, RELIABILITY, SECURITY
  --statuses <list>         OPEN, CONFIRMED, FALSE_POSITIVE, ACCEPTED, FIXED
  --tags <list>             Comma-separated tags
  --rules <list>            Rule keys (e.g., java:S1234)
  --assignee <login>        Assignee login (__me__ for current user)
  --author <email>          SCM author
  -l, --languages <list>    Languages (java, js, py, etc.)

Time filters:
  --created-after <date>    YYYY-MM-DD
  --created-before <date>   YYYY-MM-DD
  --created-in-last <span>  e.g., 1m2w (1 month 2 weeks), 7d
  --new                     Only new issues since leak period

Resolution:
  --resolved true/false     Filter by resolution
  --unresolved              Only unresolved issues

Output:
  -n, --limit <num>         Max results (default: 100, max: 500)
  --page <num>              Page number
  --sort <field>            SEVERITY, CREATION_DATE, UPDATE_DATE, etc.
  --asc / --desc            Sort direction
  --json                    JSON output
  --md, --markdown          Markdown output (with emojis and tables)
  --full                    Show all issue details (default: concise)
```

## Tips

- Set `SONAR_ORG` to avoid typing `--org` every time
- Use `--json` with `jq` for custom filtering
- Use `--new` to focus on newly introduced issues
- Combine filters: `--types BUG --severities CRITICAL,BLOCKER --unresolved`
