package domain

import "testing"

func TestNormalizeIssuesQuery(t *testing.T) {
	resolved := false
	ascending := true

	query, err := NormalizeIssuesQuery(IssuesQuery{
		Org:              "my-org",
		Project:          "my-project",
		Types:            []string{"bug, code_smell"},
		Severities:       []string{"critical"},
		ImpactSeverities: []string{"high"},
		Qualities:        []string{"security"},
		Statuses:         []string{"open"},
		CreatedAfter:     "2026-01-01",
		CreatedInLast:    "1m2w",
		Resolved:         &resolved,
		Ascending:        &ascending,
		Limit:            100,
		Page:             2,
	})
	if err != nil {
		t.Fatalf("normalize query: %v", err)
	}
	if got, want := query.Types[0], "BUG"; got != want {
		t.Fatalf("first type = %q, want %q", got, want)
	}
	if got, want := query.Types[1], "CODE_SMELL"; got != want {
		t.Fatalf("second type = %q, want %q", got, want)
	}
	if got, want := query.Severities[0], "CRITICAL"; got != want {
		t.Fatalf("severity = %q, want %q", got, want)
	}
	if got, want := query.ImpactSeverities[0], "HIGH"; got != want {
		t.Fatalf("impact severity = %q, want %q", got, want)
	}
	if got, want := query.Qualities[0], "SECURITY"; got != want {
		t.Fatalf("quality = %q, want %q", got, want)
	}
	if got, want := query.Statuses[0], "OPEN"; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := query.CreatedInLast, "1m2w"; got != want {
		t.Fatalf("created in last = %q, want %q", got, want)
	}
}

func TestNormalizeIssuesQueryRejectsBadType(t *testing.T) {
	_, err := NormalizeIssuesQuery(IssuesQuery{Org: "my-org", Types: []string{"bugs"}, Limit: 100, Page: 1})
	if err == nil {
		t.Fatal("expected invalid type error")
	}
}
