package sonar

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/LarsEckart/sonar-tool/internal/domain"
)

type Rule struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type IssueImpact struct {
	SoftwareQuality string `json:"softwareQuality"`
	Severity        string `json:"severity"`
}

type Issue struct {
	Key                        string        `json:"key"`
	Rule                       string        `json:"rule"`
	Severity                   string        `json:"severity"`
	Type                       string        `json:"type"`
	Message                    string        `json:"message"`
	Component                  string        `json:"component"`
	Project                    string        `json:"project"`
	Line                       int           `json:"line,omitzero"`
	Status                     string        `json:"status"`
	IssueStatus                string        `json:"issueStatus"`
	Assignee                   string        `json:"assignee,omitzero"`
	Author                     string        `json:"author,omitzero"`
	Tags                       []string      `json:"tags,omitzero"`
	CreationDate               string        `json:"creationDate,omitzero"`
	UpdateDate                 string        `json:"updateDate,omitzero"`
	Effort                     string        `json:"effort,omitzero"`
	CleanCodeAttribute         string        `json:"cleanCodeAttribute,omitzero"`
	CleanCodeAttributeCategory string        `json:"cleanCodeAttributeCategory,omitzero"`
	Impacts                    []IssueImpact `json:"impacts,omitzero"`
}

type IssuesPage struct {
	Paging Paging  `json:"paging"`
	Issues []Issue `json:"issues"`
	Rules  []Rule  `json:"rules"`
}

func (c *Client) ListIssuesPage(ctx context.Context, query domain.IssuesQuery) (IssuesPage, error) {
	values := url.Values{}
	if query.Project != "" {
		values.Set("componentKeys", query.Project)
	}
	if query.Org != "" {
		values.Set("organization", query.Org)
	}
	if query.Branch != "" {
		values.Set("branch", query.Branch)
	}
	if query.PullRequest != "" {
		values.Set("pullRequest", query.PullRequest)
	}
	setCSV(values, "types", query.Types)
	setCSV(values, "severities", query.Severities)
	setCSV(values, "impactSeverities", query.ImpactSeverities)
	setCSV(values, "impactSoftwareQualities", query.Qualities)
	setCSV(values, "issueStatuses", query.Statuses)
	setCSV(values, "tags", query.Tags)
	setCSV(values, "rules", query.Rules)
	if query.Assignee != "" {
		values.Set("assignees", query.Assignee)
	}
	if query.Author != "" {
		values.Set("author", query.Author)
	}
	setCSV(values, "languages", query.Languages)
	if query.CreatedAfter != "" {
		values.Set("createdAfter", query.CreatedAfter)
	}
	if query.CreatedBefore != "" {
		values.Set("createdBefore", query.CreatedBefore)
	}
	if query.CreatedInLast != "" {
		values.Set("createdInLast", query.CreatedInLast)
	}
	if query.SinceLeakPeriod {
		values.Set("sinceLeakPeriod", "true")
	}
	if query.Resolved != nil {
		values.Set("resolved", strconv.FormatBool(*query.Resolved))
	}
	if query.Sort != "" {
		values.Set("s", query.Sort)
	}
	if query.Ascending != nil {
		values.Set("asc", strconv.FormatBool(*query.Ascending))
	}
	values.Set("ps", strconv.Itoa(query.Limit))
	values.Set("p", strconv.Itoa(query.Page))
	values.Set("additionalFields", "rules")

	var response IssuesPage
	if err := c.do(ctx, http.MethodGet, "/api/issues/search", values, &response); err != nil {
		return IssuesPage{}, err
	}
	return response, nil
}

func (c *Client) ListIssuesAll(ctx context.Context, query domain.IssuesQuery) ([]Issue, []Rule, Paging, error) {
	pageQuery := query
	pageQuery.Page = 1

	issues := []Issue{}
	rulesByKey := map[string]Rule{}
	lastPaging := Paging{PageIndex: 1, PageSize: query.Limit}

	for {
		page, err := c.ListIssuesPage(ctx, pageQuery)
		if err != nil {
			return nil, nil, Paging{}, err
		}

		issues = append(issues, page.Issues...)
		for _, rule := range page.Rules {
			rulesByKey[rule.Key] = rule
		}
		lastPaging = page.Paging

		if len(issues) >= page.Paging.Total || len(page.Issues) == 0 {
			break
		}

		pageQuery.Page++
	}

	return issues, sortedRules(rulesByKey), lastPaging, nil
}

func setCSV(values url.Values, key string, items []string) {
	if len(items) == 0 {
		return
	}
	values.Set(key, strings.Join(items, ","))
}

func sortedRules(rulesByKey map[string]Rule) []Rule {
	if len(rulesByKey) == 0 {
		return nil
	}
	keys := slices.Sorted(maps.Keys(rulesByKey))
	rules := make([]Rule, 0, len(keys))
	for _, key := range keys {
		rules = append(rules, rulesByKey[key])
	}
	return rules
}

func IssueLocation(issue Issue) string {
	component := issue.Component
	if before, after, found := strings.Cut(component, ":"); found {
		component = after
		if before == "" {
			component = issue.Component
		}
	}
	if issue.Line > 0 {
		return fmt.Sprintf("%s:%d", component, issue.Line)
	}
	return component
}
