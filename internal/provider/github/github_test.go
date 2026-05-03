package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v68/github"
	ghprovider "github.com/sevenam/markdown-backlog-sync/internal/provider/github"

	"github.com/sevenam/markdown-backlog-sync/internal/provider"
)

// fakeCreds implements provider.CredentialResolver returning a fixed token.
type fakeCreds struct{ token string }

func (f fakeCreds) Resolve(string) (string, error) { return f.token, nil }

// newTestServer registers handler under the given path prefix and returns a
// provider wired to that server.
func newTestServer(t *testing.T, mux *http.ServeMux) (provider.Provider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	// We use NewWithClient to inject a test-server-pointed HTTP client.
	httpClient := &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport},
	}
	client := gogithub.NewClient(httpClient).WithAuthToken("test-token")
	prov := ghprovider.NewWithClient("gh-test", client, "owner", "repo")
	return prov, srv
}

// rewriteTransport rewrites the scheme+host of every outgoing request to
// point at the test server.
type rewriteTransport struct {
	base  string // e.g. "http://127.0.0.1:12345"
	inner http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.URL.Scheme = "http"
	r.URL.Host = req.URL.Host
	// Replace whatever host was set with the test server address.
	parsed, _ := http.NewRequest("GET", rt.base, nil)
	r.URL.Scheme = parsed.URL.Scheme
	r.URL.Host = parsed.URL.Host
	return rt.inner.RoundTrip(r)
}

// ---- helpers ----------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ---- tests ------------------------------------------------------------------

func TestList_ReturnsIssues(t *testing.T) {
	mux := http.NewServeMux()
	updated := mustTime("2024-03-01T10:00:00Z")
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "all" {
			t.Errorf("expected state=all, got %s", r.URL.Query().Get("state"))
		}
		issues := []map[string]any{
			{
				"number":     1,
				"title":      "First issue",
				"body":       "body text",
				"state":      "open",
				"html_url":   "https://github.com/owner/repo/issues/1",
				"updated_at": updated.Format(time.RFC3339),
				"assignees":  []map[string]any{{"login": "alice"}},
				"labels":     []map[string]any{{"name": "bug"}},
			},
			{
				// Pull requests are included by GitHub — must be filtered out.
				"number":       2,
				"title":        "A pull request",
				"state":        "open",
				"html_url":     "https://github.com/owner/repo/pull/2",
				"updated_at":   updated.Format(time.RFC3339),
				"pull_request": map[string]any{"url": "x"},
			},
		}
		writeJSON(w, issues)
	})

	prov, _ := newTestServer(t, mux)
	items, err := prov.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 issue (PR filtered), got %d", len(items))
	}
	it := items[0]
	if it.ID != "1" {
		t.Errorf("ID = %q, want %q", it.ID, "1")
	}
	if it.Title != "First issue" {
		t.Errorf("Title = %q", it.Title)
	}
	if it.State != "open" {
		t.Errorf("State = %q", it.State)
	}
	if len(it.Assignees) != 1 || it.Assignees[0] != "alice" {
		t.Errorf("Assignees = %v", it.Assignees)
	}
	if len(it.Labels) != 1 || it.Labels[0] != "bug" {
		t.Errorf("Labels = %v", it.Labels)
	}
}

func TestList_SinceFilter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		since := r.URL.Query().Get("since")
		if since == "" {
			t.Error("expected since parameter to be set")
		}
		writeJSON(w, []any{})
	})

	prov, _ := newTestServer(t, mux)
	since := mustTime("2024-01-01T00:00:00Z")
	_, err := prov.List(context.Background(), &since)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet_ReturnsIssue(t *testing.T) {
	mux := http.NewServeMux()
	updated := mustTime("2024-03-01T10:00:00Z")
	mux.HandleFunc("/repos/owner/repo/issues/42", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"number":     42,
			"title":      "The answer",
			"body":       "body",
			"state":      "closed",
			"html_url":   "https://github.com/owner/repo/issues/42",
			"updated_at": updated.Format(time.RFC3339),
		})
	})

	prov, _ := newTestServer(t, mux)
	it, err := prov.Get(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != "42" {
		t.Errorf("ID = %q", it.ID)
	}
	if it.State != "closed" {
		t.Errorf("State = %q", it.State)
	}
}

func TestGet_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/99", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]any{"message": "Not Found"})
	})

	prov, _ := newTestServer(t, mux)
	_, err := prov.Get(context.Background(), "99")
	if !errors.Is(err, provider.ErrItemNotFound) {
		t.Errorf("want ErrItemNotFound, got %v", err)
	}
}

func TestGet_PRIsItemNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/5", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"number":       5,
			"title":        "PR",
			"state":        "open",
			"html_url":     "https://github.com/owner/repo/pull/5",
			"updated_at":   "2024-01-01T00:00:00Z",
			"pull_request": map[string]any{"url": "x"},
		})
	})

	prov, _ := newTestServer(t, mux)
	_, err := prov.Get(context.Background(), "5")
	if !errors.Is(err, provider.ErrItemNotFound) {
		t.Errorf("want ErrItemNotFound, got %v", err)
	}
}

func TestCreate_ReturnsNewItem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]any{
			"number":     7,
			"title":      req["title"],
			"body":       req["body"],
			"state":      "open",
			"html_url":   "https://github.com/owner/repo/issues/7",
			"updated_at": "2024-06-01T12:00:00Z",
		})
	})

	prov, _ := newTestServer(t, mux)
	created, err := prov.Create(context.Background(), provider.RemoteItem{
		Title: "New issue",
		Body:  "Some body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "7" {
		t.Errorf("ID = %q", created.ID)
	}
	if created.Title != "New issue" {
		t.Errorf("Title = %q", created.Title)
	}
}

func TestUpdate_PatchesIssue(t *testing.T) {
	updated := "2024-07-01T00:00:00Z"
	mux := http.NewServeMux()
	// Initial Get for optimistic concurrency check.
	mux.HandleFunc("/repos/owner/repo/issues/10", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, map[string]any{
				"number":     10,
				"title":      "Original",
				"state":      "open",
				"html_url":   "https://github.com/owner/repo/issues/10",
				"updated_at": updated,
			})
			return
		}
		// PATCH
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]any{
			"number":     10,
			"title":      req["title"],
			"state":      "closed",
			"html_url":   "https://github.com/owner/repo/issues/10",
			"updated_at": "2024-07-02T00:00:00Z",
		})
	})

	prov, _ := newTestServer(t, mux)
	newTitle := "Updated title"
	newState := "closed"
	baseRev := "2024-07-01T00:00:00Z"
	result, err := prov.Update(context.Background(), "10", provider.ItemPatch{
		Title: &newTitle,
		State: &newState,
	}, baseRev)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "closed" {
		t.Errorf("State = %q", result.State)
	}
}

func TestUpdate_ConflictDetected(t *testing.T) {
	mux := http.NewServeMux()
	// Return a different updated_at than the baseRev the caller will pass.
	mux.HandleFunc("/repos/owner/repo/issues/11", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"number":     11,
			"title":      "Something",
			"state":      "open",
			"html_url":   "x",
			"updated_at": "2024-09-01T00:00:00Z", // newer than baseRev below
		})
	})

	prov, _ := newTestServer(t, mux)
	title := "new"
	_, err := prov.Update(context.Background(), "11", provider.ItemPatch{Title: &title}, "2024-08-01T00:00:00Z")
	if !errors.Is(err, provider.ErrConflict) {
		t.Errorf("want ErrConflict, got %v", err)
	}
}

func TestAuthError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]any{"message": "Bad credentials"})
	})

	prov, _ := newTestServer(t, mux)
	_, err := prov.List(context.Background(), nil)
	if !errors.Is(err, provider.ErrAuth) {
		t.Errorf("want ErrAuth, got %v", err)
	}
}

func TestRateLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "9999999999")
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]any{"message": "API rate limit exceeded"})
	})

	prov, _ := newTestServer(t, mux)
	_, err := prov.List(context.Background(), nil)
	if !errors.Is(err, provider.ErrRateLimited) {
		t.Errorf("want ErrRateLimited, got %v", err)
	}
}

func TestCapabilities(t *testing.T) {
	prov := ghprovider.NewWithClient("test", nil, "o", "r")
	caps := prov.Capabilities()
	if !caps.Labels {
		t.Error("Labels capability should be true")
	}
	if !caps.Milestones {
		t.Error("Milestones capability should be true")
	}
	if !caps.DeltaSince {
		t.Error("DeltaSince capability should be true")
	}
	if caps.HardDelete {
		t.Error("HardDelete should be false (GitHub cannot delete issues via API)")
	}
}
