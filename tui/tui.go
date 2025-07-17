// Package tui implements the tui for the app
package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"quattrinitrack/logger"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fc595f")).
			Background(lipgloss.Color("#252525")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FFFDF5"))

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a63c40"))

	menuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Padding(1, 2)

	selectedMenuStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fc595f")).
				Background(lipgloss.Color("#252525")).
				Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 2)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#fc595f")).
			Padding(1, 2)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	focusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fc595f"))

	noStyle = lipgloss.NewStyle()
)

type screen int

const (
	menuScreen screen = iota
	logsScreen
	authScreen
)

type authMode int

const (
	loginMode authMode = iota
	registerMode
)

type keyMap struct {
	up    key.Binding
	down  key.Binding
	quit  key.Binding
	help  key.Binding
	clear key.Binding
	enter key.Binding
	back  key.Binding
	tab   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.help, k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down, k.enter},
		{k.back, k.clear, k.quit},
	}
}

var keys = keyMap{
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	clear: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear logs"),
	),
	enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back to menu"),
	),
	tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
}

type menuItem struct {
	title       string
	description string
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

type model struct {
	viewport      viewport.Model
	ready         bool
	showHelp      bool
	lastUpdate    time.Time
	currentScreen screen
	menuItems     []menuItem
	selectedItem  int
	width         int
	height        int

	// Auth fields
	authMode      authMode
	emailInput    textinput.Model
	passwordInput textinput.Model
	focusedInput  int
	authMessage   string
	authToken     string
	isLoggedIn    bool
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentScreen {
		case menuScreen:
			switch {
			case key.Matches(msg, keys.quit):
				return m, tea.Quit
			case key.Matches(msg, keys.help):
				m.showHelp = !m.showHelp
			case key.Matches(msg, keys.up):
				if m.selectedItem > 0 {
					m.selectedItem--
				}
			case key.Matches(msg, keys.down):
				if m.selectedItem < len(m.menuItems)-1 {
					m.selectedItem++
				}
			case key.Matches(msg, keys.enter):
				switch m.selectedItem {
				case 0: // View Logs
					m.currentScreen = logsScreen
					if m.ready {
						m.viewport.SetContent(m.formatLogs())
					}
				case 1: // Login/Register
					m.currentScreen = authScreen
					m.authMode = loginMode
					m.authMessage = ""
					m.focusedInput = 0
					m.emailInput.Focus()
					m.passwordInput.Blur()
				case 2: // Exit
					return m, tea.Quit
				}
			}

		case logsScreen:
			switch {
			case key.Matches(msg, keys.quit):
				return m, tea.Quit
			case key.Matches(msg, keys.back):
				m.currentScreen = menuScreen
			case key.Matches(msg, keys.help):
				m.showHelp = !m.showHelp
			case key.Matches(msg, keys.clear):
				logger.ClearLogs()
				m.viewport.SetContent(m.formatLogs())
			default:
				// Handle viewport navigation
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}

		case authScreen:
			switch {
			case key.Matches(msg, keys.quit):
				return m, tea.Quit
			case msg.String() == "esc":
				// Only handle escape key for going back, not backspace
				m.currentScreen = menuScreen
			case key.Matches(msg, keys.tab):
				m.focusedInput = (m.focusedInput + 1) % 2
				if m.focusedInput == 0 {
					m.emailInput.Focus()
					m.passwordInput.Blur()
				} else {
					m.emailInput.Blur()
					m.passwordInput.Focus()
				}
			case key.Matches(msg, keys.enter):
				// Submit auth request
				email := m.emailInput.Value()
				password := m.passwordInput.Value()

				if email == "" || password == "" {
					m.authMessage = "Email and password are required"
					break
				}

				// Perform auth request
				err := m.performAuth(email, password)
				if err != nil {
					m.authMessage = fmt.Sprintf("Error: %v", err)
				} else {
					if m.authMode == loginMode {
						m.authMessage = "Login successful!"
						m.isLoggedIn = true
					} else {
						m.authMessage = "Registration successful!"
					}
				}
			case msg.String() == "ctrl+r":
				// Toggle between login and register (use Ctrl+R to avoid conflicts)
				if m.authMode == loginMode {
					m.authMode = registerMode
				} else {
					m.authMode = loginMode
				}
				m.authMessage = ""
			default:
				// Update text inputs for all other keys (including 'r' and backspace)
				if m.focusedInput == 0 {
					m.emailInput, cmd = m.emailInput.Update(msg)
					cmds = append(cmds, cmd)
				} else {
					m.passwordInput, cmd = m.passwordInput.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.formatLogs())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case tickMsg:
		// Update logs every second if we're on the logs screen
		if m.currentScreen == logsScreen && m.ready {
			m.viewport.SetContent(m.formatLogs())
		}
		m.lastUpdate = time.Time(msg)
		cmds = append(cmds, tick())
	}

	return m, tea.Batch(cmds...)
}

func (m *model) performAuth(email, password string) error {
	authReq := authRequest{
		Email:    email,
		Password: password,
	}

	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return err
	}

	var endpoint string
	if m.authMode == loginMode {
		endpoint = "http://localhost:8080/login"
	} else {
		endpoint = "http://localhost:8080/register"
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	if m.authMode == loginMode {
		var authResp authResponse
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return err
		}
		m.authToken = authResp.Token
	}

	return nil
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	switch m.currentScreen {
	case menuScreen:
		return m.menuView()
	case logsScreen:
		return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	case authScreen:
		return m.authView()
	default:
		return m.menuView()
	}
}

func (m model) menuView() string {
	var s strings.Builder

	// Header
	title := titleStyle.Render("QuattriniTrack - Main Menu")
	s.WriteString(title + "\n\n")

	// Login status
	if m.isLoggedIn {
		s.WriteString(successStyle.Render("✓ Logged in") + "\n\n")
	}

	// Menu items
	for i, item := range m.menuItems {
		cursor := "  "
		if i == m.selectedItem {
			cursor = "▶ "
			s.WriteString(selectedMenuStyle.Render(cursor + item.title))
		} else {
			s.WriteString(menuItemStyle.Render(cursor + item.title))
		}
		s.WriteString("\n")

		// Add description for selected item
		if i == m.selectedItem {
			s.WriteString(menuItemStyle.Render("    " + item.description))
			s.WriteString("\n")
		}
	}

	// Help text
	s.WriteString("\n")
	if m.showHelp {
		s.WriteString("↑/k: move up • ↓/j: move down • enter: select • ?: toggle help • q: quit\n")
	} else {
		s.WriteString("Press ? for help\n")
	}

	return s.String()
}

func (m model) authView() string {
	var s strings.Builder

	// Header
	var title string
	if m.authMode == loginMode {
		title = titleStyle.Render("QuattriniTrack - Login")
	} else {
		title = titleStyle.Render("QuattriniTrack - Register")
	}
	s.WriteString(title + "\n\n")

	// Email input
	s.WriteString("Email:\n")
	if m.focusedInput == 0 {
		s.WriteString(inputStyle.Render(m.emailInput.View()))
	} else {
		s.WriteString(noStyle.Render(m.emailInput.View()))
	}
	s.WriteString("\n\n")

	// Password input
	s.WriteString("Password:\n")
	if m.focusedInput == 1 {
		s.WriteString(inputStyle.Render(m.passwordInput.View()))
	} else {
		s.WriteString(noStyle.Render(m.passwordInput.View()))
	}
	s.WriteString("\n\n")

	// Auth message
	if m.authMessage != "" {
		if strings.Contains(m.authMessage, "successful") {
			s.WriteString(successStyle.Render(m.authMessage))
		} else {
			s.WriteString(errorStyle.Render(m.authMessage))
		}
		s.WriteString("\n\n")
	}

	// Instructions
	s.WriteString("Tab: next field • Enter: submit • Ctrl+R: toggle login/register • Esc: back to menu\n")

	if m.authMode == loginMode {
		s.WriteString("Press 'Ctrl+R' to switch to register mode\n")
	} else {
		s.WriteString("Press 'Ctrl+R' to switch to login mode\n")
	}

	return s.String()
}

func (m model) headerView() string {
	title := titleStyle.Render("QuattriniTrack - Server Logs")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))

	helpText := ""
	if m.showHelp {
		helpText = "\n" + "↑/k: scroll up • ↓/j: scroll down • c: clear logs • esc: back to menu • q: quit • ?: toggle help"
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, line, info) + helpText
}

func (m model) formatLogs() string {
	logs := logger.GetLogs()

	if len(logs) == 0 {
		return logStyle.Render("No logs yet... Server is running and waiting for requests.")
	}

	var content strings.Builder

	for _, log := range logs {
		timestamp := timestampStyle.Render(log.Timestamp.Format("15:04:05"))
		level := log.Level

		// Clean up the message (remove newlines and extra spaces)
		message := strings.TrimSpace(log.Message)

		// Style based on content
		var styledMessage string
		if strings.Contains(strings.ToLower(message), "error") ||
			strings.Contains(strings.ToLower(message), "failed") ||
			strings.Contains(strings.ToLower(message), "panic") {
			styledMessage = errorStyle.Render(message)
		} else {
			styledMessage = logStyle.Render(message)
		}

		content.WriteString(fmt.Sprintf("%s [%s] %s\n", timestamp, level, styledMessage))
	}

	return content.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Init() {
	// Enable log suppression so logs don't appear in console
	logger.SetSuppress(true)

	// Initialize text inputs
	emailInput := textinput.New()
	emailInput.Placeholder = "Enter your email"
	emailInput.CharLimit = 50
	emailInput.Width = 30

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Enter your password"
	passwordInput.CharLimit = 50
	passwordInput.Width = 30
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'

	// Initialize menu items
	menuItems := []menuItem{
		{
			title:       "View Logs",
			description: "View real-time server logs with scrolling and filtering",
		},
		{
			title:       "Login/Register",
			description: "Authenticate with your credentials or create a new account",
		},
		{
			title:       "Exit",
			description: "Close the application",
		},
	}

	p := tea.NewProgram(
		model{
			lastUpdate:    time.Now(),
			currentScreen: menuScreen,
			menuItems:     menuItems,
			selectedItem:  0,
			emailInput:    emailInput,
			passwordInput: passwordInput,
			focusedInput:  0,
			authMode:      loginMode,
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
	}

	// Restore original output when TUI exits
	logger.RestoreOriginalOutput()
}
