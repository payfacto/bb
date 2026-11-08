package bitbucket_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// userWorkspacesHandler serves GET /user/workspaces (slug/uuid only, as the
// real API does) and GET /workspaces/{slug} detail lookups.
func userWorkspacesHandler(t *testing.T, details map[string]map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/workspaces":
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{
				{"type": "workspace_access", "administrator": true,
					"workspace": map[string]any{"slug": "acme", "uuid": "{w-1}"}},
				{"type": "workspace_access", "administrator": false,
					"workspace": map[string]any{"slug": "other", "uuid": "{w-2}"}},
			}})
		default:
			slug := strings.TrimPrefix(r.URL.Path, "/workspaces/")
			detail, ok := details[slug]
			if !ok {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			mustEncodeJSON(t, w, detail)
		}
	}
}

func TestWorkspaces_List_UsesUserWorkspacesAndUnwrapsAccess(t *testing.T) {
	client := newTestClient(t, userWorkspacesHandler(t, map[string]map[string]any{
		"acme":  {"slug": "acme", "name": "Acme", "uuid": "{w-1}"},
		"other": {"slug": "other", "name": "Other", "uuid": "{w-2}"},
	}))
	got, err := client.Workspaces().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Slug != "acme" || got[1].Name != "Other" {
		t.Fatalf("unexpected workspaces: %+v", got)
	}
	if !got[0].Administrator || got[1].Administrator {
		t.Errorf("administrator flag not carried over: %+v", got)
	}
}

func TestWorkspaces_List_KeepsBaseEntryWhenDetailLookupFails(t *testing.T) {
	client := newTestClient(t, userWorkspacesHandler(t, map[string]map[string]any{
		"acme": {"slug": "acme", "name": "Acme", "uuid": "{w-1}"},
	}))
	got, err := client.Workspaces().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Slug != "other" || got[1].Name != "" {
		t.Fatalf("expected base entry for 'other', got %+v", got)
	}
}

func TestWorkspaces_List_ReturnsErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/workspaces" {
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{
				{"workspace": map[string]any{"slug": "acme"}},
				{"workspace": map[string]any{"slug": "other"}},
			}})
			return
		}
		cancel()
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	if _, err := client.Workspaces().List(ctx); err == nil {
		t.Fatal("expected an error after context cancellation, got slug-only results")
	}
}
