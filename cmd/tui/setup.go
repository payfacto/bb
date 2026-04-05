package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	setupFieldWorkspace = iota
	setupFieldRepo
	setupFieldUsername
	setupFieldPassword
	setupFieldCount
)

// setupModel is the TUI setup wizard for first-run or reconfiguration.
type setupModel struct {
	fields   []textinput.Model
	focus    int
	cfgPath  string
	existing *config.Config
	err       error
	done      bool
	message   string
	newClient *bitbucket.Client
	newCfg    *config.Config
}

func newSetupView(cfgPath string, existing *config.Config) *setupModel {
	if existing == nil {
		existing = &config.Config{}
	}

	fields := make([]textinput.Model, setupFieldCount)

	fields[setupFieldWorkspace] = textinput.New()
	fields[setupFieldWorkspace].Placeholder = "workspace slug"
	fields[setupFieldWorkspace].SetValue(existing.Workspace)
	fields[setupFieldWorkspace].CharLimit = 100
	fields[setupFieldWorkspace].Focus()

	fields[setupFieldRepo] = textinput.New()
	fields[setupFieldRepo].Placeholder = "repo slug (optional)"
	fields[setupFieldRepo].SetValue(existing.Repo)
	fields[setupFieldRepo].CharLimit = 100

	fields[setupFieldUsername] = textinput.New()
	fields[setupFieldUsername].Placeholder = "username or email"
	fields[setupFieldUsername].SetValue(existing.Username)
	fields[setupFieldUsername].CharLimit = 100

	fields[setupFieldPassword] = textinput.New()
	fields[setupFieldPassword].Placeholder = "app password"
	fields[setupFieldPassword].EchoMode = textinput.EchoPassword
	fields[setupFieldPassword].EchoCharacter = '*'
	fields[setupFieldPassword].CharLimit = 200

	return &setupModel{
		fields:   fields,
		cfgPath:  cfgPath,
		existing: existing,
	}
}

func (m *setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		// After save, any key transitions to home menu
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, func() tea.Msg {
				return rebuildMenuMsg{client: m.newClient, cfg: m.newCfg}
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			return m, m.nextField()
		case tea.KeyShiftTab, tea.KeyUp:
			return m, m.prevField()
		case tea.KeyEnter:
			if m.focus == setupFieldCount-1 {
				return m, m.save()
			}
			return m, m.nextField()
		case tea.KeyEscape:
			// Only allow escape if we have existing valid config
			if m.existing.Workspace != "" && m.existing.Username != "" {
				return m, popView
			}
		}
	case saveResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.done = true
		m.message = msg.message
		m.newClient = msg.client
		m.newCfg = msg.cfg
		return m, nil
	}

	// Forward to focused field
	var cmd tea.Cmd
	m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
	return m, cmd
}

func (m *setupModel) nextField() tea.Cmd {
	m.fields[m.focus].Blur()
	m.focus = (m.focus + 1) % setupFieldCount
	m.fields[m.focus].Focus()
	return textinput.Blink
}

func (m *setupModel) prevField() tea.Cmd {
	m.fields[m.focus].Blur()
	m.focus = (m.focus - 1 + setupFieldCount) % setupFieldCount
	m.fields[m.focus].Focus()
	return textinput.Blink
}

type saveResultMsg struct {
	err     error
	message string
	client  *bitbucket.Client
	cfg     *config.Config
}

// rebuildMenuMsg tells the app to replace the stack with a new home menu.
type rebuildMenuMsg struct {
	client *bitbucket.Client
	cfg    *config.Config
}

func (m *setupModel) save() tea.Cmd {
	ws := m.fields[setupFieldWorkspace].Value()
	repo := m.fields[setupFieldRepo].Value()
	user := m.fields[setupFieldUsername].Value()
	pass := m.fields[setupFieldPassword].Value()

	if ws == "" || user == "" {
		return func() tea.Msg {
			return saveResultMsg{err: fmt.Errorf("workspace and username are required")}
		}
	}

	existing := m.existing
	cfgPath := m.cfgPath

	return func() tea.Msg {
		authType := existing.AuthType
		if pass != "" {
			authType = "apppassword"
		}

		updated := &config.Config{
			Workspace:     ws,
			Repo:          repo,
			Username:      user,
			AuthType:      authType,
			OAuthClientID: existing.OAuthClientID,
		}
		if err := updated.Save(cfgPath); err != nil {
			return saveResultMsg{err: fmt.Errorf("save config: %w", err)}
		}

		if pass != "" {
			if err := auth.SetToken(user, pass); err != nil {
				return saveResultMsg{err: fmt.Errorf("store token in keyring: %w", err)}
			}
		}

		// Resolve token for client creation
		tok := pass
		if tok == "" {
			t, err := auth.GetToken(user)
			if err == nil {
				tok = t
			}
		}

		if tok == "" {
			return saveResultMsg{err: fmt.Errorf("no token available (enter app password or run 'bb auth login')")}
		}

		updated.Token = tok
		client := bitbucket.New(updated)
		if updated.HasOAuth() {
			client.SetBearerToken(tok)
		}

		return saveResultMsg{
			message: fmt.Sprintf("Config saved to %s", cfgPath),
			client:  client,
			cfg:     updated,
		}
	}
}

func (m *setupModel) View() string {
	var sb strings.Builder

	sb.WriteString(headerStyle.Render("bb setup"))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render("Configure your Bitbucket Cloud connection"))
	sb.WriteString("\n")
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", 50)))
	sb.WriteString("\n\n")

	labels := []string{"Workspace", "Default repo", "Username", "App password"}
	for i, label := range labels {
		if i == m.focus {
			sb.WriteString(helpKeyStyle.Render(fmt.Sprintf("  %-14s ", label)))
		} else {
			sb.WriteString(subtitleStyle.Render(fmt.Sprintf("  %-14s ", label)))
		}
		sb.WriteString(m.fields[i].View())
		sb.WriteString("\n")
	}

	if m.err != nil {
		sb.WriteString("\n")
		sb.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n")
	}

	if m.done {
		sb.WriteString("\n")
		sb.WriteString(successStyle.Render("  " + m.message))
		sb.WriteString("\n")
		sb.WriteString(subtitleStyle.Render("  Press any key to continue..."))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m *setupModel) Title() string { return "Setup" }
func (m *setupModel) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab/↓", "next field")),
		key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab/↑", "prev field")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
	}
}
