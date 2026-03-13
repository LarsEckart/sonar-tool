package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lars/sonar-tool/internal/auth"
	configpath "github.com/lars/sonar-tool/internal/config"
	"github.com/lars/sonar-tool/internal/domain"
	"github.com/lars/sonar-tool/internal/sonar"
)

func TestRecordLiveFixtures(t *testing.T) {
	if os.Getenv("SONAR_RECORD_FIXTURES") != "1" {
		t.Skip("set SONAR_RECORD_FIXTURES=1 to refresh sanitized live fixtures")
	}

	project := strings.TrimSpace(os.Getenv("SONAR_FIXTURE_PROJECT"))
	if project == "" {
		project = "example-project"
	}

	configPath, err := configpath.ConfigFilePath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	store := auth.NewStore(configPath, auth.OSSecretStore{})
	resolved, err := store.Resolve("", "")
	if err != nil {
		t.Fatalf("resolve auth: %v", err)
	}

	timeout := 30
	if value := strings.TrimSpace(os.Getenv("SONAR_TIMEOUT")); value != "" {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			t.Fatalf("parse SONAR_TIMEOUT: %v", parseErr)
		}
		timeout = parsed
	}

	client, err := sonar.NewClient(resolved.Host, resolved.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	projectsPage, err := client.ListProjectsPage(t.Context(), resolved.Org, 1, 5)
	if err != nil {
		t.Fatalf("list projects page: %v", err)
	}

	issuesQuery, err := domain.NormalizeIssuesQuery(domain.IssuesQuery{Org: resolved.Org, Project: project, Limit: 3, Page: 1})
	if err != nil {
		t.Fatalf("normalize issues query: %v", err)
	}
	issuesPage, err := client.ListIssuesPage(t.Context(), issuesQuery)
	if err != nil {
		t.Fatalf("list issues page: %v", err)
	}

	fixturesDir := filepath.Join(moduleRoot(t), "tests", "testdata")
	if err := os.MkdirAll(fixturesDir, 0o755); err != nil {
		t.Fatalf("create fixtures dir: %v", err)
	}

	if err := writeJSONFixture(filepath.Join(fixturesDir, "auth_validate.json"), map[string]bool{"valid": true}); err != nil {
		t.Fatalf("write auth validate fixture: %v", err)
	}
	if err := writeJSONFixture(filepath.Join(fixturesDir, "organizations_search.json"), sanitizeOrganizationsResponse()); err != nil {
		t.Fatalf("write organizations fixture: %v", err)
	}
	if err := writeJSONFixture(filepath.Join(fixturesDir, "projects_page_1.json"), sanitizeProjectsPage(projectsPage)); err != nil {
		t.Fatalf("write projects fixture: %v", err)
	}
	if err := writeJSONFixture(filepath.Join(fixturesDir, "issues_page_1.json"), sanitizeIssuesPage(issuesPage)); err != nil {
		t.Fatalf("write issues fixture: %v", err)
	}
}

func sanitizeOrganizationsResponse() map[string]any {
	return map[string]any{
		"organizations": []map[string]string{{
			"key":  "example-org",
			"name": "Example Organization",
		}},
		"paging": map[string]int{
			"pageIndex": 1,
			"pageSize":  1,
			"total":     1,
		},
	}
}

func sanitizeProjectsPage(page sonar.ProjectsPage) sonar.ProjectsPage {
	sanitized := sonar.ProjectsPage{
		Paging:     sonar.Paging{PageIndex: 1, PageSize: len(page.Components), Total: len(page.Components) + 7},
		Components: make([]sonar.Project, 0, len(page.Components)),
	}
	for index := range page.Components {
		sanitized.Components = append(sanitized.Components, sonar.Project{
			Key:  fmt.Sprintf("project-%03d", index+1),
			Name: fmt.Sprintf("Example Project %03d", index+1),
		})
	}
	return sanitized
}

func sanitizeIssuesPage(page sonar.IssuesPage) sonar.IssuesPage {
	sanitized := sonar.IssuesPage{
		Paging: sonar.Paging{PageIndex: 1, PageSize: len(page.Issues), Total: len(page.Issues) + 9},
		Issues: make([]sonar.Issue, 0, len(page.Issues)),
		Rules:  make([]sonar.Rule, 0, len(page.Rules)),
	}
	for index, issue := range page.Issues {
		sanitized.Issues = append(sanitized.Issues, sonar.Issue{
			Key:         fmt.Sprintf("issue-%03d", index+1),
			Rule:        issue.Rule,
			Severity:    issue.Severity,
			Type:        issue.Type,
			Message:     fmt.Sprintf("Example issue message %03d.", index+1),
			Component:   fmt.Sprintf("example-org:module-%03d/src/example/File%03d%s", index+1, index+1, fileExtension(issue.Component)),
			Project:     "project-001",
			Line:        10 + index,
			Status:      fallbackValue(issue.Status, "OPEN"),
			IssueStatus: fallbackValue(issue.IssueStatus, fallbackValue(issue.Status, "OPEN")),
			Assignee:    fmt.Sprintf("user%03d@example.test", index+1),
			Author:      fmt.Sprintf("author%03d@example.test", index+1),
			Tags:        issue.Tags,
			Effort:      issue.Effort,
			CreationDate: func() string {
				return fmt.Sprintf("2026-01-%02dT10:00:00+0000", index+1)
			}(),
			UpdateDate: func() string {
				return fmt.Sprintf("2026-01-%02dT11:00:00+0000", index+1)
			}(),
			CleanCodeAttribute:         issue.CleanCodeAttribute,
			CleanCodeAttributeCategory: issue.CleanCodeAttributeCategory,
			Impacts:                    issue.Impacts,
		})
	}

	for index, rule := range page.Rules {
		sanitized.Rules = append(sanitized.Rules, sonar.Rule{
			Key:  rule.Key,
			Name: fmt.Sprintf("Example rule name %03d", index+1),
		})
	}

	return sanitized
}

func fileExtension(component string) string {
	lastSlash := strings.LastIndex(component, "/")
	segment := component
	if lastSlash >= 0 {
		segment = component[lastSlash+1:]
	}
	if dot := strings.LastIndex(segment, "."); dot >= 0 {
		return segment[dot:]
	}
	return ".txt"
}

func fallbackValue(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func writeJSONFixture(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
