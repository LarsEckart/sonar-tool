package format

import (
	"fmt"
	"strings"

	"github.com/LarsEckart/sonar-tool/internal/sonar"
)

func IssuesPlain(issues []sonar.Issue, rules []sonar.Rule, paging sonar.Paging, full bool, all bool) string {
	header := fmt.Sprintf("Found %d issues (showing %d on page %d)", paging.Total, len(issues), paging.PageIndex)
	if all {
		header = fmt.Sprintf("Found %d issues (showing all %d)", paging.Total, len(issues))
	}
	lines := []string{header, ""}

	ruleNames := ruleNameMap(rules)
	for index, issue := range issues {
		issueNumber := index + 1 + (paging.PageIndex-1)*paging.PageSize
		if full {
			lines = append(lines, issueFull(issueNumber, issue, ruleNames))
		} else {
			lines = append(lines, issueConcise(issueNumber, issue, ruleNames))
		}
	}

	if !all && paging.Total > paging.PageIndex*paging.PageSize {
		remaining := paging.Total - paging.PageIndex*paging.PageSize
		lines = append(lines, fmt.Sprintf("... %d more issues. Use --page %d or --all.", remaining, paging.PageIndex+1))
	}

	return strings.Join(lines, "\n")
}

func IssuesMarkdown(issues []sonar.Issue, rules []sonar.Rule, paging sonar.Paging, all bool) string {
	summary := fmt.Sprintf("> Found **%d** issues (showing %d on page %d)", paging.Total, len(issues), paging.PageIndex)
	if all {
		summary = fmt.Sprintf("> Found **%d** issues (showing all %d)", paging.Total, len(issues))
	}
	lines := []string{"# Sonar Issues Report", "", summary, ""}

	if len(issues) > 0 {
		lines = append(lines, issueSummary(issues)...)
		lines = append(lines, "---", "")
	}

	lines = append(lines, "## Issues", "")
	ruleNames := ruleNameMap(rules)
	for index, issue := range issues {
		issueNumber := index + 1 + (paging.PageIndex-1)*paging.PageSize
		lines = append(lines, issueMarkdown(issueNumber, issue, ruleNames))
	}

	if !all && paging.Total > paging.PageIndex*paging.PageSize {
		remaining := paging.Total - paging.PageIndex*paging.PageSize
		lines = append(lines, "---", "", fmt.Sprintf("*%d more issues. Use `--page %d` or `--all`.*", remaining, paging.PageIndex+1))
	}

	return strings.Join(lines, "\n")
}

func issueConcise(issueNumber int, issue sonar.Issue, ruleNames map[string]string) string {
	lines := []string{fmt.Sprintf("%d. %s %s  %s", issueNumber, fallback(issue.Severity, impactSeverity(issue)), fallback(issue.Type, "ISSUE"), sonar.IssueLocation(issue))}
	lines = append(lines, fmt.Sprintf("   Rule: %s%s", issue.Rule, optionalRuleName(ruleNames[issue.Rule])))
	lines = append(lines, fmt.Sprintf("   Message: %s", issue.Message), "")
	return strings.Join(lines, "\n")
}

func issueFull(issueNumber int, issue sonar.Issue, ruleNames map[string]string) string {
	lines := []string{fmt.Sprintf("%d. %s", issueNumber, issue.Message)}
	lines = append(lines, fmt.Sprintf("   Rule: %s%s", issue.Rule, optionalRuleName(ruleNames[issue.Rule])))
	lines = append(lines, fmt.Sprintf("   Severity: %s", fallback(issue.Severity, impactSeverity(issue))))
	lines = append(lines, fmt.Sprintf("   Type: %s", fallback(issue.Type, "N/A")))
	lines = append(lines, fmt.Sprintf("   Status: %s", fallback(issueStatus(issue), "N/A")))
	lines = append(lines, fmt.Sprintf("   File: %s", sonar.IssueLocation(issue)))
	if issue.Effort != "" {
		lines = append(lines, fmt.Sprintf("   Effort: %s", issue.Effort))
	}
	if len(issue.Impacts) > 0 {
		lines = append(lines, fmt.Sprintf("   Impacts: %s", joinImpacts(issue.Impacts)))
	}
	if len(issue.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("   Tags: %s", strings.Join(issue.Tags, ", ")))
	}
	if issue.Assignee != "" {
		lines = append(lines, fmt.Sprintf("   Assignee: %s", issue.Assignee))
	}
	if issue.Author != "" {
		lines = append(lines, fmt.Sprintf("   Author: %s", issue.Author))
	}
	if issue.CreationDate != "" {
		lines = append(lines, fmt.Sprintf("   Created: %s", issue.CreationDate))
	}
	if issue.UpdateDate != "" && issue.UpdateDate != issue.CreationDate {
		lines = append(lines, fmt.Sprintf("   Updated: %s", issue.UpdateDate))
	}
	lines = append(lines, fmt.Sprintf("   Key: %s", issue.Key), "")
	return strings.Join(lines, "\n")
}

func issueMarkdown(issueNumber int, issue sonar.Issue, ruleNames map[string]string) string {
	lines := []string{fmt.Sprintf("### %d. %s", issueNumber, issue.Message), "", fmt.Sprintf("- **Rule:** `%s`%s", issue.Rule, optionalRuleName(ruleNames[issue.Rule])), fmt.Sprintf("- **Severity:** %s", fallback(issue.Severity, impactSeverity(issue))), fmt.Sprintf("- **Type:** %s", fallback(issue.Type, "N/A")), fmt.Sprintf("- **Status:** %s", fallback(issueStatus(issue), "N/A")), fmt.Sprintf("- **File:** `%s`", sonar.IssueLocation(issue))}
	if len(issue.Impacts) > 0 {
		lines = append(lines, fmt.Sprintf("- **Impacts:** %s", joinImpacts(issue.Impacts)))
	}
	if issue.Assignee != "" {
		lines = append(lines, fmt.Sprintf("- **Assignee:** %s", issue.Assignee))
	}
	if issue.Author != "" {
		lines = append(lines, fmt.Sprintf("- **Author:** %s", issue.Author))
	}
	if len(issue.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("- **Tags:** %s", strings.Join(issue.Tags, ", ")))
	}
	lines = append(lines, fmt.Sprintf("- **Key:** `%s`", issue.Key), "")
	return strings.Join(lines, "\n")
}

func issueSummary(issues []sonar.Issue) []string {
	bySeverity := map[string]int{}
	byType := map[string]int{}
	for _, issue := range issues {
		severity := fallback(issue.Severity, impactSeverity(issue))
		bySeverity[severity]++
		byType[fallback(issue.Type, "N/A")]++
	}

	lines := []string{"## Summary", ""}
	severityOrder := []string{"BLOCKER", "CRITICAL", "MAJOR", "MINOR", "INFO", "HIGH", "MEDIUM", "LOW"}
	severityParts := []string{}
	for _, severity := range severityOrder {
		if count := bySeverity[severity]; count > 0 {
			severityParts = append(severityParts, fmt.Sprintf("%s: %d", severity, count))
		}
	}
	if len(severityParts) > 0 {
		lines = append(lines, "**By Severity:** "+strings.Join(severityParts, " | "), "")
	}
	if len(byType) > 0 {
		typeParts := []string{}
		for _, issueType := range []string{"BUG", "VULNERABILITY", "CODE_SMELL", "SECURITY_HOTSPOT", "N/A"} {
			if count := byType[issueType]; count > 0 {
				typeParts = append(typeParts, fmt.Sprintf("%s: %d", issueType, count))
			}
		}
		lines = append(lines, "**By Type:** "+strings.Join(typeParts, " | "), "")
	}
	return lines
}

func ruleNameMap(rules []sonar.Rule) map[string]string {
	result := make(map[string]string, len(rules))
	for _, rule := range rules {
		result[rule.Key] = rule.Name
	}
	return result
}

func optionalRuleName(name string) string {
	if name == "" {
		return ""
	}
	return " - " + name
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func issueStatus(issue sonar.Issue) string {
	if issue.IssueStatus != "" {
		return issue.IssueStatus
	}
	return issue.Status
}

func impactSeverity(issue sonar.Issue) string {
	if len(issue.Impacts) == 0 {
		return "N/A"
	}
	return issue.Impacts[0].Severity
}

func joinImpacts(impacts []sonar.IssueImpact) string {
	parts := make([]string, 0, len(impacts))
	for _, impact := range impacts {
		parts = append(parts, fmt.Sprintf("%s:%s", impact.SoftwareQuality, impact.Severity))
	}
	return strings.Join(parts, ", ")
}
