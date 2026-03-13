package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lars/sonar-tool/internal/domain"
	"github.com/lars/sonar-tool/internal/format"
	"github.com/lars/sonar-tool/internal/sonar"
	cli "github.com/urfave/cli/v3"
)

func newIssuesCommand() *cli.Command {
	return &cli.Command{
		Name:  "issues",
		Usage: "List Sonar issues",
		Commands: []*cli.Command{
			{
				Name:        "list",
				Usage:       "List issues for a project or organization",
				Description: "Prints concise plain text by default, markdown for reports, or a stable JSON envelope for scripts.",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Sonar project key"},
					&cli.StringFlag{Name: "branch", Usage: "Branch name"},
					&cli.StringFlag{Name: "pull-request", Aliases: []string{"pr"}, Usage: "Pull request ID"},
					&cli.StringSliceFlag{Name: "types", Usage: "Issue types (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringSliceFlag{Name: "severities", Usage: "Legacy severities (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringSliceFlag{Name: "impact-severities", Usage: "Impact severities (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringSliceFlag{Name: "qualities", Usage: "Software qualities (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringSliceFlag{Name: "statuses", Usage: "Issue statuses (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringSliceFlag{Name: "tags", Usage: "Tags (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringSliceFlag{Name: "rules", Usage: "Rule keys (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringFlag{Name: "assignee", Usage: "Assignee login"},
					&cli.StringFlag{Name: "author", Usage: "SCM author"},
					&cli.StringSliceFlag{Name: "languages", Usage: "Languages (csv or repeated)", Config: cli.StringConfig{TrimSpace: true}},
					&cli.StringFlag{Name: "created-after", Usage: "Created after date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "created-before", Usage: "Created before date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "created-in-last", Usage: "Created in last span like 7d or 1m2w"},
					&cli.BoolFlag{Name: "new", Usage: "Only issues from the current leak period"},
					&cli.BoolFlag{Name: "resolved", Usage: "Filter by resolution status; use --resolved=false to request unresolved issues"},
					&cli.BoolFlag{Name: "unresolved", Usage: "Shortcut for --resolved=false"},
					&cli.StringFlag{Name: "sort", Usage: "Sonar sort field"},
					&cli.BoolFlag{Name: "asc", Usage: "Sort ascending"},
					&cli.BoolFlag{Name: "desc", Usage: "Sort descending"},
					&cli.IntFlag{Name: "limit", Usage: "Issues per page", Value: 100},
					&cli.IntFlag{Name: "page", Usage: "1-based page index", Value: 1},
					&cli.BoolFlag{Name: "all", Usage: "Fetch all pages"},
					&cli.BoolFlag{Name: "full", Usage: "Include more issue detail in plain output"},
				},
				Action: issuesListAction,
			},
		},
	}
}

type issuesListEnvelope struct {
	Query struct {
		Org     string `json:"org,omitzero"`
		Project string `json:"project,omitzero"`
		Page    int    `json:"page"`
		Limit   int    `json:"limit"`
		All     bool   `json:"all"`
	} `json:"query"`
	Paging struct {
		PageIndex int `json:"page_index"`
		PageSize  int `json:"page_size"`
		Total     int `json:"total"`
	} `json:"paging"`
	Issues []sonar.Issue `json:"issues"`
	Rules  []sonar.Rule  `json:"rules,omitzero"`
}

func issuesListAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("asc") && cmd.Bool("desc") {
		return usageError("--asc and --desc cannot be used together", "choose one sort direction")
	}
	if cmd.IsSet("resolved") && cmd.Bool("resolved") && cmd.Bool("unresolved") {
		return usageError("--resolved and --unresolved cannot conflict", "use either --resolved=true or --unresolved")
	}

	ascending := sortDirection(cmd)
	resolved := resolvedFilter(cmd)

	resolvedAuth, err := resolveAuth(cmd, false)
	if err != nil {
		return err
	}

	query, err := domain.NormalizeIssuesQuery(domain.IssuesQuery{
		Org:              resolvedAuth.Org,
		Project:          cmd.String("project"),
		Branch:           cmd.String("branch"),
		PullRequest:      cmd.String("pull-request"),
		Types:            cmd.StringSlice("types"),
		Severities:       cmd.StringSlice("severities"),
		ImpactSeverities: cmd.StringSlice("impact-severities"),
		Qualities:        cmd.StringSlice("qualities"),
		Statuses:         cmd.StringSlice("statuses"),
		Tags:             cmd.StringSlice("tags"),
		Rules:            cmd.StringSlice("rules"),
		Assignee:         cmd.String("assignee"),
		Author:           cmd.String("author"),
		Languages:        cmd.StringSlice("languages"),
		CreatedAfter:     cmd.String("created-after"),
		CreatedBefore:    cmd.String("created-before"),
		CreatedInLast:    cmd.String("created-in-last"),
		SinceLeakPeriod:  cmd.Bool("new"),
		Resolved:         resolved,
		Sort:             cmd.String("sort"),
		Ascending:        ascending,
		Limit:            cmd.Int("limit"),
		Page:             cmd.Int("page"),
		All:              cmd.Bool("all"),
	})
	if err != nil {
		return usageError(err.Error(), "use --help for valid issue filters")
	}

	client, err := newSonarClientFromResolvedAuth(cmd, resolvedAuth)
	if err != nil {
		return err
	}

	var issues []sonar.Issue
	var rules []sonar.Rule
	var paging sonar.Paging
	if query.All {
		issues, rules, paging, err = client.ListIssuesAll(ctx, query)
	} else {
		var page sonar.IssuesPage
		page, err = client.ListIssuesPage(ctx, query)
		issues = page.Issues
		rules = page.Rules
		paging = page.Paging
	}
	if err != nil {
		return mapSonarError(err)
	}

	envelope := issuesListEnvelope{Issues: issues, Rules: rules}
	envelope.Query.Org = query.Org
	envelope.Query.Project = query.Project
	envelope.Query.Page = query.Page
	envelope.Query.Limit = query.Limit
	envelope.Query.All = query.All
	envelope.Paging.PageIndex = paging.PageIndex
	envelope.Paging.PageSize = paging.PageSize
	envelope.Paging.Total = paging.Total

	mode, err := outputMode(cmd)
	if err != nil {
		return err
	}

	switch mode {
	case OutputModeJSON:
		encoder := json.NewEncoder(cmd.Root().Writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(envelope)
	case OutputModeMarkdown:
		_, err = fmt.Fprintln(cmd.Root().Writer, format.IssuesMarkdown(issues, rules, paging, query.All))
		return err
	default:
		_, err = fmt.Fprintln(cmd.Root().Writer, format.IssuesPlain(issues, rules, paging, cmd.Bool("full"), query.All))
		return err
	}
}

func sortDirection(cmd *cli.Command) *bool {
	if cmd.Bool("asc") {
		return new(true)
	}
	if cmd.Bool("desc") {
		return new(false)
	}
	return nil
}

func resolvedFilter(cmd *cli.Command) *bool {
	if cmd.Bool("unresolved") {
		return new(false)
	}
	if cmd.IsSet("resolved") {
		return new(cmd.Bool("resolved"))
	}
	return nil
}
