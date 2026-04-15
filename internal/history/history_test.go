package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	h, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil History")
	}
	if h.Favourites == nil {
		t.Error("expected non-nil Favourites map")
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hist.json")
	h := &History{Favourites: map[string][]string{"ws": {"repo-a"}}}
	h.MRU = []MRUEntry{{Workspace: "ws", Slug: "repo-a", Name: "Repo A"}}

	if err := h.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Favourites["ws"]) != 1 || loaded.Favourites["ws"][0] != "repo-a" {
		t.Errorf("unexpected favourites: %v", loaded.Favourites)
	}
	if len(loaded.MRU) != 1 || loaded.MRU[0].Slug != "repo-a" {
		t.Errorf("unexpected MRU: %v", loaded.MRU)
	}
}

func TestLoadBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hist.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestToggleFavourite(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}

	h.ToggleFavourite("ws", "repo-b")
	if !h.IsFavourite("ws", "repo-b") {
		t.Error("expected repo-b to be a favourite after first toggle")
	}

	h.ToggleFavourite("ws", "repo-a")
	// favourites should be sorted
	if h.Favourites["ws"][0] != "repo-a" || h.Favourites["ws"][1] != "repo-b" {
		t.Errorf("expected sorted favourites, got %v", h.Favourites["ws"])
	}

	h.ToggleFavourite("ws", "repo-b")
	if h.IsFavourite("ws", "repo-b") {
		t.Error("expected repo-b removed after second toggle")
	}
	if !h.IsFavourite("ws", "repo-a") {
		t.Error("expected repo-a still a favourite")
	}
}

func TestAddMRU(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}

	for i, slug := range []string{"a", "b", "c", "d", "e"} {
		h.AddMRU("ws", slug, "Repo "+slug)
		if h.MRU[0].Slug != slug {
			t.Errorf("step %d: expected newest slug %q at index 0, got %q", i, slug, h.MRU[0].Slug)
		}
	}
	if len(h.MRU) != 5 {
		t.Errorf("expected MRU capped at 5, got %d", len(h.MRU))
	}

	// Adding a 6th should evict oldest.
	h.AddMRU("ws", "f", "Repo f")
	if len(h.MRU) != 5 {
		t.Errorf("expected MRU still capped at 5, got %d", len(h.MRU))
	}
	if h.MRU[0].Slug != "f" {
		t.Errorf("expected f at front, got %q", h.MRU[0].Slug)
	}
	// "a" should be gone
	for _, e := range h.MRU {
		if e.Slug == "a" {
			t.Error("expected 'a' evicted from MRU")
		}
	}

	// Re-visiting existing entry moves it to front.
	h.AddMRU("ws", "c", "Repo c")
	if h.MRU[0].Slug != "c" {
		t.Errorf("expected c moved to front, got %q", h.MRU[0].Slug)
	}
	if len(h.MRU) != 5 {
		t.Errorf("expected length unchanged after dedupe, got %d", len(h.MRU))
	}
}

func TestRecentSlugs(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	h.AddMRU("ws1", "a", "A")
	h.AddMRU("ws2", "x", "X")
	h.AddMRU("ws1", "b", "B")

	slugs := h.RecentSlugs("ws1")
	if len(slugs) != 2 {
		t.Fatalf("expected 2 slugs for ws1, got %d", len(slugs))
	}
	if slugs[0] != "b" || slugs[1] != "a" {
		t.Errorf("expected [b a], got %v", slugs)
	}
}

func TestHistoryPath(t *testing.T) {
	base := filepath.Join("home", "user")
	p := HistoryPath(filepath.Join(base, ".bbcloud.yaml"))
	want := filepath.Join(base, ".bbcloud_history.json")
	if p != want {
		t.Errorf("expected %q, got %q", want, p)
	}
}

func TestRepoCacheSetGet(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}

	// Empty cache returns false.
	if _, ok := h.Repos("ws"); ok {
		t.Error("expected no cached repos for fresh History")
	}

	repos := []CachedRepo{
		{Slug: "repo-a", Name: "Repo A", IsPrivate: false},
		{Slug: "repo-b", Name: "Repo B", IsPrivate: true},
	}
	h.SetRepos("ws", repos)

	got, ok := h.Repos("ws")
	if !ok {
		t.Fatal("expected cached repos after SetRepos")
	}
	if len(got) != 2 || got[0].Slug != "repo-a" || got[1].Slug != "repo-b" {
		t.Errorf("unexpected cached repos: %v", got)
	}

	// Different workspace is independent.
	if _, ok := h.Repos("other-ws"); ok {
		t.Error("expected no cached repos for different workspace")
	}
}

func TestRepoCacheEmptySlice(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	// Storing an empty slice must still report as a cache miss.
	h.SetRepos("ws", []CachedRepo{})
	if _, ok := h.Repos("ws"); ok {
		t.Error("expected cache miss for empty repo slice")
	}
}

func TestRepoCacheClear(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	h.SetRepos("ws", []CachedRepo{{Slug: "repo-a", Name: "Repo A"}})

	h.ClearRepos("ws")
	if _, ok := h.Repos("ws"); ok {
		t.Error("expected no cached repos after ClearRepos")
	}
}

func TestRepoCachePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hist.json")
	h := &History{Favourites: make(map[string][]string)}
	h.SetRepos("ws", []CachedRepo{{Slug: "repo-a", Name: "Repo A", IsPrivate: true}})

	if err := h.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := loaded.Repos("ws")
	if !ok || len(got) != 1 || got[0].Slug != "repo-a" || !got[0].IsPrivate {
		t.Errorf("unexpected repos after round-trip: ok=%v repos=%v", ok, got)
	}
}

// --- Project favourites ---

func TestToggleProjectFavourite(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}

	h.ToggleProjectFavourite("ws", "PROJ-B")
	if !h.IsProjectFavourite("ws", "PROJ-B") {
		t.Error("expected PROJ-B to be a favourite after first toggle")
	}

	h.ToggleProjectFavourite("ws", "PROJ-A")
	// favourites should be sorted
	if h.ProjectFavourites["ws"][0] != "PROJ-A" || h.ProjectFavourites["ws"][1] != "PROJ-B" {
		t.Errorf("expected sorted project favourites, got %v", h.ProjectFavourites["ws"])
	}

	h.ToggleProjectFavourite("ws", "PROJ-B")
	if h.IsProjectFavourite("ws", "PROJ-B") {
		t.Error("expected PROJ-B removed after second toggle")
	}
	if !h.IsProjectFavourite("ws", "PROJ-A") {
		t.Error("expected PROJ-A still a favourite")
	}
}

func TestIsProjectFavourite_NilMap(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	// ProjectFavourites is nil — must not panic.
	if h.IsProjectFavourite("ws", "X") {
		t.Error("expected false on nil ProjectFavourites map")
	}
}

// --- Project MRU ---

func TestAddProjectMRU(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}

	for i, key := range []string{"A", "B", "C", "D", "E"} {
		h.AddProjectMRU("ws", key, "Project "+key)
		if h.ProjectMRU[0].Key != key {
			t.Errorf("step %d: expected newest key %q at index 0, got %q", i, key, h.ProjectMRU[0].Key)
		}
	}
	if len(h.ProjectMRU) != 5 {
		t.Errorf("expected ProjectMRU capped at 5, got %d", len(h.ProjectMRU))
	}

	// Adding a 6th should evict oldest.
	h.AddProjectMRU("ws", "F", "Project F")
	if len(h.ProjectMRU) != 5 {
		t.Errorf("expected ProjectMRU still capped at 5, got %d", len(h.ProjectMRU))
	}
	if h.ProjectMRU[0].Key != "F" {
		t.Errorf("expected F at front, got %q", h.ProjectMRU[0].Key)
	}
	for _, e := range h.ProjectMRU {
		if e.Key == "A" {
			t.Error("expected 'A' evicted from ProjectMRU")
		}
	}

	// Re-visiting existing entry moves it to front.
	h.AddProjectMRU("ws", "C", "Project C")
	if h.ProjectMRU[0].Key != "C" {
		t.Errorf("expected C moved to front, got %q", h.ProjectMRU[0].Key)
	}
	if len(h.ProjectMRU) != 5 {
		t.Errorf("expected length unchanged after dedupe, got %d", len(h.ProjectMRU))
	}
}

func TestRecentProjectKeys(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	h.AddProjectMRU("ws1", "X", "X")
	h.AddProjectMRU("ws2", "Y", "Y")
	h.AddProjectMRU("ws1", "Z", "Z")

	keys := h.RecentProjectKeys("ws1")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys for ws1, got %d", len(keys))
	}
	if keys[0] != "Z" || keys[1] != "X" {
		t.Errorf("expected [Z X], got %v", keys)
	}
}

// --- Project cache ---

func TestProjectCacheSetGet(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}

	if _, ok := h.Projects("ws"); ok {
		t.Error("expected no cached projects for fresh History")
	}

	projects := []CachedProject{
		{Key: "AA", Name: "Alpha", IsPrivate: true},
		{Key: "BB", Name: "Beta", IsPrivate: false},
	}
	h.SetProjects("ws", projects)

	got, ok := h.Projects("ws")
	if !ok {
		t.Fatal("expected cached projects after SetProjects")
	}
	if len(got) != 2 || got[0].Key != "AA" || got[1].Key != "BB" {
		t.Errorf("unexpected cached projects: %v", got)
	}

	if _, ok := h.Projects("other-ws"); ok {
		t.Error("expected no cached projects for different workspace")
	}
}

func TestProjectCacheEmptySlice(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	h.SetProjects("ws", []CachedProject{})
	if _, ok := h.Projects("ws"); ok {
		t.Error("expected cache miss for empty project slice")
	}
}

func TestProjectCacheClear(t *testing.T) {
	h := &History{Favourites: make(map[string][]string)}
	h.SetProjects("ws", []CachedProject{{Key: "AA", Name: "Alpha"}})

	h.ClearProjects("ws")
	if _, ok := h.Projects("ws"); ok {
		t.Error("expected no cached projects after ClearProjects")
	}
}

func TestProjectCachePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hist.json")
	h := &History{Favourites: make(map[string][]string)}
	h.SetProjects("ws", []CachedProject{{Key: "AA", Name: "Alpha", IsPrivate: true}})

	if err := h.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := loaded.Projects("ws")
	if !ok || len(got) != 1 || got[0].Key != "AA" || !got[0].IsPrivate {
		t.Errorf("unexpected projects after round-trip: ok=%v projects=%v", ok, got)
	}
}
