package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LarsEckart/sonar-tool/internal/format"
	"github.com/LarsEckart/sonar-tool/internal/sonar"
	cli "github.com/urfave/cli/v3"
)

func newProjectsCommand() *cli.Command {
	return &cli.Command{
		Name:  "projects",
		Usage: "List Sonar projects",
		Commands: []*cli.Command{
			{
				Name:        "list",
				Usage:       "List projects in an organization",
				Description: "Prints project keys in plain mode, markdown for reports, or a stable JSON envelope for scripts.",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Projects per page", Value: 100},
					&cli.IntFlag{Name: "page", Usage: "1-based page index", Value: 1},
					&cli.BoolFlag{Name: "all", Usage: "Fetch all pages"},
				},
				Action: projectsListAction,
			},
		},
	}
}

type projectsListEnvelope struct {
	Query struct {
		Org   string `json:"org"`
		Page  int    `json:"page"`
		Limit int    `json:"limit"`
		All   bool   `json:"all"`
	} `json:"query"`
	Paging struct {
		PageIndex int `json:"page_index"`
		PageSize  int `json:"page_size"`
		Total     int `json:"total"`
	} `json:"paging"`
	Projects []sonar.Project `json:"projects"`
}

func projectsListAction(ctx context.Context, cmd *cli.Command) error {
	limit := cmd.Int("limit")
	page := cmd.Int("page")
	if limit < 1 || limit > 500 {
		return usageError("invalid value for --limit", "use a number between 1 and 500")
	}
	if page < 1 {
		return usageError("invalid value for --page", "use a page number greater than or equal to 1")
	}

	allPages := cmd.Bool("all")
	if allPages {
		page = 1
	}

	resolved, err := resolveAuth(cmd, true)
	if err != nil {
		return err
	}

	client, err := newSonarClientFromResolvedAuth(cmd, resolved)
	if err != nil {
		return err
	}

	var projects []sonar.Project
	var paging sonar.Paging
	if allPages {
		projects, paging, err = client.ListProjectsAll(ctx, resolved.Org, limit)
	} else {
		var response sonar.ProjectsPage
		response, err = client.ListProjectsPage(ctx, resolved.Org, page, limit)
		projects = response.Components
		paging = response.Paging
	}
	if err != nil {
		return mapSonarError(err)
	}

	envelope := projectsListEnvelope{Projects: projects}
	envelope.Query.Org = resolved.Org
	envelope.Query.Page = page
	envelope.Query.Limit = limit
	envelope.Query.All = allPages
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
		_, err = fmt.Fprintln(cmd.Root().Writer, format.ProjectsMarkdown(projects, paging, allPages))
		return err
	default:
		_, err = fmt.Fprintln(cmd.Root().Writer, format.ProjectsPlain(projects, paging, allPages))
		return err
	}
}
