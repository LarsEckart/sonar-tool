package format

import (
	"fmt"
	"strings"

	"github.com/lars/sonar-tool/internal/sonar"
)

func ProjectsPlain(projects []sonar.Project, paging sonar.Paging, all bool) string {
	lines := make([]string, 0, len(projects)+1)
	for _, project := range projects {
		lines = append(lines, project.Key)
	}

	if !all && paging.Total > paging.PageIndex*paging.PageSize {
		remaining := paging.Total - paging.PageIndex*paging.PageSize
		lines = append(lines, fmt.Sprintf("# ... %d more. Use --page %d or --all.", remaining, paging.PageIndex+1))
	}

	return strings.Join(lines, "\n")
}

func ProjectsMarkdown(projects []sonar.Project, paging sonar.Paging, all bool) string {
	lines := []string{"## Projects", ""}
	for _, project := range projects {
		lines = append(lines, fmt.Sprintf("- `%s`", project.Key))
	}

	if !all && paging.Total > paging.PageIndex*paging.PageSize {
		remaining := paging.Total - paging.PageIndex*paging.PageSize
		lines = append(lines, "", fmt.Sprintf("*%d more projects. Use `--page %d` or `--all`.*", remaining, paging.PageIndex+1))
	}

	return strings.Join(lines, "\n")
}
