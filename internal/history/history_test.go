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
	p := HistoryPath("/home/user/.bbcloud.yaml")
	want := "/home/user/.bbcloud_history.json"
	if p != want {
		t.Errorf("expected %q, got %q", want, p)
	}
}
