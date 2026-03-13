package sonar

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Paging struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
	Total     int `json:"total"`
}

type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ProjectsPage struct {
	Paging     Paging    `json:"paging"`
	Components []Project `json:"components"`
}

func (c *Client) ListProjectsPage(ctx context.Context, org string, page, limit int) (ProjectsPage, error) {
	query := url.Values{}
	query.Set("organization", org)
	query.Set("p", fmt.Sprintf("%d", page))
	query.Set("ps", fmt.Sprintf("%d", limit))

	var response ProjectsPage
	if err := c.do(ctx, http.MethodGet, "/api/components/search_projects", query, &response); err != nil {
		return ProjectsPage{}, err
	}
	return response, nil
}

func (c *Client) ListProjectsAll(ctx context.Context, org string, limit int) ([]Project, Paging, error) {
	pageIndex := 1
	projects := []Project{}
	lastPaging := Paging{PageIndex: 1, PageSize: limit}

	for {
		page, err := c.ListProjectsPage(ctx, org, pageIndex, limit)
		if err != nil {
			return nil, Paging{}, err
		}

		projects = append(projects, page.Components...)
		lastPaging = page.Paging

		if len(projects) >= page.Paging.Total || len(page.Components) == 0 {
			break
		}

		pageIndex++
	}

	return projects, lastPaging, nil
}
