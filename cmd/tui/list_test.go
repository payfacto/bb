package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- applySearch ---

func TestApplySearch_Empty(t *testing.T) {
	m := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	m.items = []listItem{
		{id: "1", title: "Alpha"},
		{id: "2", title: "Beta"},
	}
	m.search.SetValue("")
	m.applySearch()

	if len(m.filtered) != 2 {
		t.Errorf("empty query: expected all 2 items, got %d", len(m.filtered))
	}
}

func TestApplySearch_MatchTitle(t *testing.T) {
	m := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	m.items = []listItem{
		{id: "1", title: "Alpha"},
		{id: "2", title: "Beta"},
		{id: "3", title: "alphanumeric"},
	}
	m.search.SetValue("alpha")
	m.applySearch()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 matches for 'alpha', got %d", len(m.filtered))
	}
}

func TestApplySearch_MatchSubtitle(t *testing.T) {
	m := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	m.items = []listItem{
		{title: "Repo A", subtitle: "main"},
		{title: "Repo B", subtitle: "develop"},
	}
	m.search.SetValue("develop")
	m.applySearch()

	if len(m.filtered) != 1 || m.filtered[0].title != "Repo B" {
		t.Errorf("expected Repo B, got %v", m.filtered)
	}
}

func TestApplySearch_MatchID(t *testing.T) {
	m := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	m.items = []listItem{
		{id: "#42", title: "Fix bug"},
		{id: "#99", title: "Add feature"},
	}
	m.search.SetValue("42")
	m.applySearch()

	if len(m.filtered) != 1 || m.filtered[0].id != "#42" {
		t.Errorf("expected item #42, got %v", m.filtered)
	}
}

func TestApplySearch_NoMatch(t *testing.T) {
	m := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	m.items = []listItem{{title: "Alpha"}, {title: "Beta"}}
	m.search.SetValue("zzz")
	m.applySearch()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.filtered))
	}
}

func TestApplySearch_CursorResets(t *testing.T) {
	m := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	m.items = []listItem{{title: "Alpha"}, {title: "Beta"}, {title: "Gamma"}}
	m.cursor = 2
	m.search.SetValue("beta")
	m.applySearch()

	if m.cursor != 0 {
		t.Errorf("expected cursor reset to 0 after search, got %d", m.cursor)
	}
}

// --- OnKey receives filtered[cursor], not items[cursor] ---
//
// Regression: before the fix, OnKey was called with items[cursor] which pointed
// to the wrong item when a search filter was active.

func TestOnKey_ReceivesFilteredItem(t *testing.T) {
	var gotSelected listItem
	cfg := ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
		OnKey: func(msg tea.KeyMsg, selected listItem, items []listItem) ([]listItem, tea.Cmd) {
			gotSelected = selected
			return nil, nil
		},
	}

	m := newListView(cfg)
	m.loading = false
	// items[0]=Apple, items[1]=Banana, items[2]=Cherry
	m.items = []listItem{
		{id: "1", title: "Apple"},
		{id: "2", title: "Banana"},
		{id: "3", title: "Cherry"},
	}
	// filter to only "Banana" and "Cherry"; cursor=1 → Cherry
	m.search.SetValue("a") // matches Apple and Banana
	m.applySearch()
	// filtered = [Apple, Banana]; move cursor to 1 (Banana)
	m.cursor = 1

	m.updateNavigation(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	if gotSelected.id != "2" {
		t.Errorf("OnKey should receive filtered[cursor] (Banana/id=2), got id=%q title=%q",
			gotSelected.id, gotSelected.title)
	}
}
