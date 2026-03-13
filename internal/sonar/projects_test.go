package sonar

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListProjectsAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization header = %q, want %q", got, "Bearer test-token")
		}
		if r.URL.Path != "/api/components/search_projects" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/components/search_projects")
		}

		page := r.URL.Query().Get("p")
		switch page {
		case "1":
			fmt.Fprint(w, `{"paging":{"pageIndex":1,"pageSize":2,"total":3},"components":[{"key":"proj-a","name":"Project A"},{"key":"proj-b","name":"Project B"}]}`)
		case "2":
			fmt.Fprint(w, `{"paging":{"pageIndex":2,"pageSize":2,"total":3},"components":[{"key":"proj-c","name":"Project C"}]}`)
		default:
			t.Fatalf("unexpected page %q", page)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	projects, paging, err := client.ListProjectsAll(t.Context(), "my-org", 2)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("project count = %d, want 3", len(projects))
	}
	if projects[2].Key != "proj-c" {
		t.Fatalf("third project key = %q, want %q", projects[2].Key, "proj-c")
	}
	if paging.Total != 3 {
		t.Fatalf("paging total = %d, want 3", paging.Total)
	}
}
