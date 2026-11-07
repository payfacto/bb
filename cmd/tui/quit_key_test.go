package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/payfacto/bb/internal/history"
)

func typeRune(model tea.Model, r rune) (tea.Model, tea.Cmd) {
	return model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

func newLoadedListApp() (tea.Model, *listModel) {
	list := newListView(ListConfig{
		Fetch: func(_ context.Context, _ string) ([]listItem, error) { return nil, nil },
	})
	list.loading = false
	list.items = []listItem{{id: "1", title: "quarkus-api"}, {id: "2", title: "billing"}}
	list.filtered = list.items
	return newApp(list, &history.History{}, newListCache()), list
}

// A 'q' typed into the list search box must filter the list, not quit the app.
func TestQuitKeyDoesNotFireWhileSearchingList(t *testing.T) {
	model, list := newLoadedListApp()

	model, _ = typeRune(model, '/')
	model, cmd := typeRune(model, 'q')

	if isQuit(cmd) {
		t.Fatal("'q' triggered quit while the list search box was focused")
	}
	if got := list.search.Value(); got != "q" {
		t.Fatalf("search value = %q, want %q", got, "q")
	}
	if len(list.filtered) != 1 || list.filtered[0].title != "quarkus-api" {
		t.Fatalf("filtered = %v, want only quarkus-api", list.filtered)
	}
}

// Outside search mode, 'q' on a list still quits the app.
func TestQuitKeyStillQuitsListWhenNotSearching(t *testing.T) {
	model, _ := newLoadedListApp()

	_, cmd := typeRune(model, 'q')

	if !isQuit(cmd) {
		t.Fatal("'q' did not quit from a list that is not searching")
	}
}

// A 'q' typed into a text prompt (e.g. new branch name) must land in the field.
func TestQuitKeyDoesNotFireInInputView(t *testing.T) {
	input := newInputView("New branch", "name", func(string) tea.Cmd { return nil })
	var model tea.Model = newApp(input, &history.History{}, newListCache())

	model, cmd := typeRune(model, 'q')

	if isQuit(cmd) {
		t.Fatal("'q' triggered quit while an input view was focused")
	}
	if got := input.input.Value(); got != "q" {
		t.Fatalf("input value = %q, want %q", got, "q")
	}
}
