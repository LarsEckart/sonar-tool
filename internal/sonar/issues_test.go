package sonar

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lars/sonar-tool/internal/domain"
)

func TestListIssuesAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issues/search" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/issues/search")
		}
		if got, want := r.URL.Query().Get("organization"), "my-org"; got != want {
			t.Fatalf("organization = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("componentKeys"), "my-project"; got != want {
			t.Fatalf("componentKeys = %q, want %q", got, want)
		}

		switch r.URL.Query().Get("p") {
		case "1":
			fmt.Fprint(w, `{"paging":{"pageIndex":1,"pageSize":2,"total":3},"issues":[{"key":"i1","rule":"go:S1","message":"first","component":"my-org:src/a.go","line":10,"severity":"MAJOR","type":"BUG"},{"key":"i2","rule":"go:S2","message":"second","component":"my-org:src/b.go","line":20,"severity":"CRITICAL","type":"CODE_SMELL"}],"rules":[{"key":"go:S1","name":"Rule One"},{"key":"go:S2","name":"Rule Two"}]}`)
		case "2":
			fmt.Fprint(w, `{"paging":{"pageIndex":2,"pageSize":2,"total":3},"issues":[{"key":"i3","rule":"go:S2","message":"third","component":"my-org:src/c.go","line":30,"severity":"MINOR","type":"BUG"}],"rules":[{"key":"go:S2","name":"Rule Two"}]}`)
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("p"))
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	issues, rules, paging, err := client.ListIssuesAll(t.Context(), domain.IssuesQuery{Org: "my-org", Project: "my-project", Limit: 2, All: true})
	if err != nil {
		t.Fatalf("list issues all: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("issue count = %d, want 3", len(issues))
	}
	if len(rules) != 2 {
		t.Fatalf("rule count = %d, want 2", len(rules))
	}
	if paging.Total != 3 {
		t.Fatalf("paging total = %d, want 3", paging.Total)
	}
	if got, want := IssueLocation(issues[0]), "src/a.go:10"; got != want {
		t.Fatalf("issue location = %q, want %q", got, want)
	}
}
