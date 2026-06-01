package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/internal/history"
)

func newSetupApp() (*appModel, *setupModel) {
	setup := newSetupView("/tmp/x.yaml", &config.Config{})
	for setup.focus != setupFieldPassword {
		setup.nextField()
	}
	return newApp(setup, &history.History{}, newListCache()), setup
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// A 'q' typed into a focused setup field must reach the field, not quit the app.
func TestQuitKeyDoesNotFireWhileTypingInSetup(t *testing.T) {
	app, _ := newSetupApp()

	var model tea.Model = app
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if isQuit(cmd) {
		t.Fatal("'q' triggered quit while a setup text field was focused")
	}
	got := model.(*appModel).stack.Top().(*setupModel).fields[setupFieldPassword].Value()
	if got != "q" {
		t.Fatalf("password field = %q, want %q ('q' should land in the field)", got, "q")
	}
}

// Pasting a token containing 'q' must land in full without quitting, whether the
// terminal delivers it rune-by-rune or as a single bracketed paste.
func TestPasteTokenWithQDoesNotQuit(t *testing.T) {
	// Synthetic token (not a real credential). It only needs to contain a 'q'
	// to exercise the global-quit-key regression and an '=' for good measure.
	token := "ATATTexample-fake-token-with-q-and-eq=DEADBEEF00"

	t.Run("rune-by-rune", func(t *testing.T) {
		app, _ := newSetupApp()
		var model tea.Model = app
		for _, r := range token {
			var cmd tea.Cmd
			model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			if isQuit(cmd) {
				t.Fatalf("quit fired while pasting rune %q", r)
			}
		}
		got := model.(*appModel).stack.Top().(*setupModel).fields[setupFieldPassword].Value()
		if got != token {
			t.Fatalf("password field = %q, want %q", got, token)
		}
	})

	t.Run("bracketed-paste", func(t *testing.T) {
		app, _ := newSetupApp()
		var model tea.Model = app
		model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(token), Paste: true})
		if isQuit(cmd) {
			t.Fatal("quit fired on bracketed paste")
		}
		got := model.(*appModel).stack.Top().(*setupModel).fields[setupFieldPassword].Value()
		if got != token {
			t.Fatalf("password field = %q, want %q", got, token)
		}
	})
}

// Ctrl+C must always quit, even while a setup text field is focused.
func TestCtrlCQuitsFromSetup(t *testing.T) {
	app, _ := newSetupApp()

	var model tea.Model = app
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !isQuit(cmd) {
		t.Fatal("ctrl+c did not quit from the setup wizard")
	}
}
