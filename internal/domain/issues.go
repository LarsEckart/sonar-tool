package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

var createdInLastPattern = regexp.MustCompile(`^(\d+[dwmy])+$`)

type IssuesQuery struct {
	Org              string
	Project          string
	Branch           string
	PullRequest      string
	Types            []string
	Severities       []string
	ImpactSeverities []string
	Qualities        []string
	Statuses         []string
	Tags             []string
	Rules            []string
	Assignee         string
	Author           string
	Languages        []string
	CreatedAfter     string
	CreatedBefore    string
	CreatedInLast    string
	SinceLeakPeriod  bool
	Resolved         *bool
	Sort             string
	Ascending        *bool
	Limit            int
	Page             int
	All              bool
}

func NormalizeIssuesQuery(raw IssuesQuery) (IssuesQuery, error) {
	query := IssuesQuery{
		Org:             strings.TrimSpace(raw.Org),
		Project:         strings.TrimSpace(raw.Project),
		Branch:          strings.TrimSpace(raw.Branch),
		PullRequest:     strings.TrimSpace(raw.PullRequest),
		Assignee:        strings.TrimSpace(raw.Assignee),
		Author:          strings.TrimSpace(raw.Author),
		CreatedAfter:    strings.TrimSpace(raw.CreatedAfter),
		CreatedBefore:   strings.TrimSpace(raw.CreatedBefore),
		CreatedInLast:   strings.TrimSpace(raw.CreatedInLast),
		Sort:            strings.TrimSpace(raw.Sort),
		SinceLeakPeriod: raw.SinceLeakPeriod,
		Resolved:        raw.Resolved,
		Ascending:       raw.Ascending,
		Limit:           raw.Limit,
		Page:            raw.Page,
		All:             raw.All,
	}

	if query.Project == "" && query.Org == "" {
		return IssuesQuery{}, fmt.Errorf("either project or organization is required")
	}
	if query.Limit < 1 || query.Limit > 500 {
		return IssuesQuery{}, fmt.Errorf("limit must be between 1 and 500")
	}
	if query.Page < 1 {
		return IssuesQuery{}, fmt.Errorf("page must be greater than or equal to 1")
	}
	if query.Ascending != nil && raw.Ascending == nil {
		query.Ascending = nil
	}

	var err error
	query.Types, err = normalizeCSVEnum(raw.Types, []string{"BUG", "CODE_SMELL", "SECURITY_HOTSPOT", "VULNERABILITY"}, true)
	if err != nil {
		return IssuesQuery{}, fmt.Errorf("invalid value for --types: %w", err)
	}
	query.Severities, err = normalizeCSVEnum(raw.Severities, []string{"INFO", "MINOR", "MAJOR", "CRITICAL", "BLOCKER"}, true)
	if err != nil {
		return IssuesQuery{}, fmt.Errorf("invalid value for --severities: %w", err)
	}
	query.ImpactSeverities, err = normalizeCSVEnum(raw.ImpactSeverities, []string{"INFO", "LOW", "MEDIUM", "HIGH", "BLOCKER"}, true)
	if err != nil {
		return IssuesQuery{}, fmt.Errorf("invalid value for --impact-severities: %w", err)
	}
	query.Qualities, err = normalizeCSVEnum(raw.Qualities, []string{"MAINTAINABILITY", "RELIABILITY", "SECURITY"}, true)
	if err != nil {
		return IssuesQuery{}, fmt.Errorf("invalid value for --qualities: %w", err)
	}
	query.Statuses, err = normalizeCSVEnum(raw.Statuses, []string{"OPEN", "CONFIRMED", "FALSE_POSITIVE", "ACCEPTED", "FIXED"}, true)
	if err != nil {
		return IssuesQuery{}, fmt.Errorf("invalid value for --statuses: %w", err)
	}
	query.Tags = normalizeCSV(raw.Tags, false)
	query.Rules = normalizeCSV(raw.Rules, false)
	query.Languages = normalizeCSV(raw.Languages, false)

	if query.CreatedAfter != "" {
		if _, err := time.Parse("2006-01-02", query.CreatedAfter); err != nil {
			return IssuesQuery{}, fmt.Errorf("invalid value for --created-after: use YYYY-MM-DD")
		}
	}
	if query.CreatedBefore != "" {
		if _, err := time.Parse("2006-01-02", query.CreatedBefore); err != nil {
			return IssuesQuery{}, fmt.Errorf("invalid value for --created-before: use YYYY-MM-DD")
		}
	}
	if query.CreatedInLast != "" && !createdInLastPattern.MatchString(strings.ToLower(query.CreatedInLast)) {
		return IssuesQuery{}, fmt.Errorf("invalid value for --created-in-last: use spans like 7d or 1m2w")
	}

	query.CreatedInLast = strings.ToLower(query.CreatedInLast)
	query.Sort = strings.ToUpper(query.Sort)
	if query.All {
		query.Page = 1
	}

	return query, nil
}

func normalizeCSVEnum(values []string, allowed []string, upper bool) ([]string, error) {
	normalized := normalizeCSV(values, upper)
	if len(normalized) == 0 {
		return nil, nil
	}

	allowedSet := map[string]struct{}{}
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}

	for _, value := range normalized {
		if _, ok := allowedSet[value]; !ok {
			return nil, fmt.Errorf("%q (allowed: %s)", value, strings.Join(allowed, ", "))
		}
	}

	return normalized, nil
}

func normalizeCSV(values []string, upper bool) []string {
	parts := []string{}
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if upper {
				trimmed = strings.ToUpper(trimmed)
			}
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	parts = slices.Compact(parts)
	return parts
}
