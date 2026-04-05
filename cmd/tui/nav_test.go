package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
)

type stubView struct{ title string }

func (v *stubView) Init() tea.Cmd                           { return nil }
func (v *stubView) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return v, nil }
func (v *stubView) View() string                            { return v.title }
func (v *stubView) Title() string                           { return v.title }
func (v *stubView) ShortHelp() []key.Binding                { return nil }

func TestNavStack_PushPop(t *testing.T) {
	var s navStack
	if s.Len() != 0 {
		t.Fatalf("expected empty stack, got %d", s.Len())
	}
	s.Push(&stubView{title: "Home"})
	s.Push(&stubView{title: "PRs"})
	if s.Len() != 2 {
		t.Fatalf("expected 2, got %d", s.Len())
	}
	top := s.Top()
	if top.Title() != "PRs" {
		t.Errorf("expected top=PRs, got %s", top.Title())
	}
	popped := s.Pop()
	if popped.Title() != "PRs" {
		t.Errorf("expected popped=PRs, got %s", popped.Title())
	}
	if s.Len() != 1 {
		t.Fatalf("expected 1 after pop, got %d", s.Len())
	}
}

func TestNavStack_PopEmpty(t *testing.T) {
	var s navStack
	v := s.Pop()
	if v != nil {
		t.Errorf("expected nil from empty pop, got %v", v)
	}
}

func TestNavStack_Breadcrumb(t *testing.T) {
	var s navStack
	s.Push(&stubView{title: "Home"})
	s.Push(&stubView{title: "Pull Requests"})
	s.Push(&stubView{title: "#42"})
	bc := s.Breadcrumb()
	for _, want := range []string{"Home", "Pull Requests", "#42"} {
		if !strings.Contains(bc, want) {
			t.Errorf("breadcrumb %q missing %q", bc, want)
		}
	}
}
