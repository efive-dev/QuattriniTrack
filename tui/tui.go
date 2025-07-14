package tui

import (
	"fmt"
	"quattrinitrack/logger"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99"))

	menuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Padding(1, 2)

	selectedMenuStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFDF5")).
				Background(lipgloss.Color("#25A065")).
				Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			Padding(0, 2)
)

type screen int

const (
	menuScreen screen = iota
	logsScreen
)

type keyMap struct {
	up    key.Binding
	down  key.Binding
	quit  key.Binding
	help  key.Binding
	clear key.Binding
	enter key.Binding
	back  key.Binding
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
}

type menuItem struct {
	title       string
	description string
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
				case 1: // About
					// Could implement about screen here
				case 2: // Settings
					// Could implement settings screen here
				case 3: // Exit
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

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	switch m.currentScreen {
	case menuScreen:
		return m.menuView()
	case logsScreen:
		return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	default:
		return m.menuView()
	}
}

func (m model) menuView() string {
	var s strings.Builder

	// Header
	title := titleStyle.Render("QuattriniTrack - Main Menu")
	s.WriteString(title + "\n\n")

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

	// Initialize menu items
	menuItems := []menuItem{
		{
			title:       "View Logs",
			description: "View real-time server logs with scrolling and filtering",
		},
		{
			title:       "About",
			description: "Information about QuattriniTrack application",
		},
		{
			title:       "Settings",
			description: "Configure application settings and preferences",
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
