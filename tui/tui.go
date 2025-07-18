// Package tui implements the tui for the app
package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"quattrinitrack/logger"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
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

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#fc595f"))
)

type screen int

const (
	menuScreen screen = iota
	logsScreen
	authScreen
	categoryScreen
	transactionScreen
)

type authMode int

const (
	loginMode authMode = iota
	registerMode
)

type categoryMode int

const (
	viewCategoriesMode categoryMode = iota
	addCategoryMode
	deleteCategoryMode
)

type keyMap struct {
	up      key.Binding
	down    key.Binding
	quit    key.Binding
	help    key.Binding
	clear   key.Binding
	enter   key.Binding
	back    key.Binding
	tab     key.Binding
	add     key.Binding
	del     key.Binding
	refresh key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.help, k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down, k.enter},
		{k.back, k.clear, k.quit},
		{k.add, k.del, k.refresh},
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
	add: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "add transaction/category"),
	),
	del: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "delete transaction/category"),
	),
	refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
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

type category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type transactionMode int

const (
	viewTransactionsMode transactionMode = iota
	addTransactionMode
	deleteTransactionMode
	filterTransactionMode
)

type transaction struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Cost         float64   `json:"cost"`
	Date         time.Time `json:"date"`
	CategoriesID int64     `json:"categoriesid"`
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

	// Category fields
	categoryMode         categoryMode
	categoryTable        table.Model
	categories           []category
	categoryInput        textinput.Model
	categoryMessage      string
	categoryIDInput      textinput.Model
	focusedCategoryInput int

	// Transaction fields
	transactionMode            transactionMode
	transactionTable           table.Model
	transactions               []transaction
	transactionInput           textinput.Model
	transactionIDInput         textinput.Model
	transactionCostInput       textinput.Model
	transactionDateInput       textinput.Model
	transactionCategoryIDInput textinput.Model
	transactionMessage         string
	transactionNameFilter      textinput.Model
	transactionDateFrom        textinput.Model
	transactionDateTo          textinput.Model
	filteredTransactions       []transaction
	focusedTransactionInput    int
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
				case 2: // Categories
					if !m.isLoggedIn {
						break
					}
					m.currentScreen = categoryScreen
					m.categoryMode = viewCategoriesMode
					m.categoryMessage = ""
					m.loadCategories()
				case 3: // Transactions
					if !m.isLoggedIn {
						break
					}
					m.currentScreen = transactionScreen
					m.transactionMode = viewTransactionsMode
					m.transactionMessage = ""
					m.loadTransactions()
				case 4: // Exit
					return m, tea.Quit
				}
			}

		case logsScreen:
			switch {
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
			case msg.String() == "esc":
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
				email := m.emailInput.Value()
				password := m.passwordInput.Value()

				if email == "" || password == "" {
					m.authMessage = "Email and password are required"
					break
				}

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
				if m.authMode == loginMode {
					m.authMode = registerMode
				} else {
					m.authMode = loginMode
				}
				m.authMessage = ""
			default:
				if m.focusedInput == 0 {
					m.emailInput, cmd = m.emailInput.Update(msg)
					cmds = append(cmds, cmd)
				} else {
					m.passwordInput, cmd = m.passwordInput.Update(msg)
					cmds = append(cmds, cmd)
				}
			}

		case categoryScreen:
			switch m.categoryMode {
			case viewCategoriesMode:
				switch {
				case key.Matches(msg, keys.back):
					m.currentScreen = menuScreen
				case key.Matches(msg, keys.add):
					m.categoryMode = addCategoryMode
					m.categoryMessage = ""
					m.categoryInput.SetValue("")
					m.categoryInput.Focus()
				case key.Matches(msg, keys.del):
					m.categoryMode = deleteCategoryMode
					m.categoryMessage = ""
					m.categoryIDInput.SetValue("")
					m.categoryIDInput.Focus()
				case key.Matches(msg, keys.refresh):
					m.loadCategories()
				case key.Matches(msg, keys.help):
					m.showHelp = !m.showHelp
				default:
					m.categoryTable, cmd = m.categoryTable.Update(msg)
					cmds = append(cmds, cmd)
				}

			case addCategoryMode:
				switch {
				case key.Matches(msg, keys.back):
					m.categoryMode = viewCategoriesMode
					m.categoryInput.Blur()
				case key.Matches(msg, keys.enter):
					name := m.categoryInput.Value()
					if name == "" {
						m.categoryMessage = "Category name is required"
						break
					}
					err := m.addCategory(name)
					if err != nil {
						m.categoryMessage = fmt.Sprintf("Error: %v", err)
					} else {
						var transactionNameFilter, transactionDateFrom, transactionDateTo textinput.Model
						transactionNameFilter = textinput.New()
						transactionNameFilter.Placeholder = "Filter by name"
						transactionNameFilter.CharLimit = 50
						transactionNameFilter.Width = 30

						transactionDateFrom = textinput.New()
						transactionDateFrom.Placeholder = "From date (YYYY-MM-DD)"
						transactionDateFrom.CharLimit = 10
						transactionDateFrom.Width = 30

						transactionDateTo = textinput.New()
						transactionDateTo.Placeholder = "To date (YYYY-MM-DD)"
						transactionDateTo.CharLimit = 10
						transactionDateTo.Width = 30

						transactionNameFilter.Focus() // Set initial focus to name filter

						m.categoryMessage = "Category added successfully!"
						m.categoryInput.SetValue("")
						m.loadCategories()
					}
				default:
					m.categoryInput, cmd = m.categoryInput.Update(msg)
					cmds = append(cmds, cmd)
				}

			case deleteCategoryMode:
				switch {
				case key.Matches(msg, keys.back):
					m.categoryMode = viewCategoriesMode
					m.categoryIDInput.Blur()
				case key.Matches(msg, keys.enter):
					idStr := m.categoryIDInput.Value()
					if idStr == "" {
						m.categoryMessage = "Category ID is required"
						break
					}
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err != nil {
						m.categoryMessage = "Invalid category ID"
						break
					}
					err = m.deleteCategory(id)
					if err != nil {
						m.categoryMessage = fmt.Sprintf("Error: %v", err)
					} else {
						m.categoryMessage = "Category deleted successfully!"
						m.categoryIDInput.SetValue("")
						m.loadCategories()
					}
				default:
					m.categoryIDInput, cmd = m.categoryIDInput.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}

		switch m.currentScreen {
		case transactionScreen:
			switch m.transactionMode {
			case viewTransactionsMode:
				// No filter inputs shown by default; shortcuts only
				switch {
				case key.Matches(msg, keys.quit):
					return m, tea.Quit
				case key.Matches(msg, keys.back):
					m.currentScreen = menuScreen
					m.transactionMode = viewTransactionsMode
				case key.Matches(msg, keys.add):
					m.transactionMode = addTransactionMode
					m.transactionMessage = ""
					m.transactionInput.SetValue("")
					m.transactionIDInput.SetValue("")
					m.transactionCostInput.SetValue("")
					m.transactionDateInput.SetValue("")
					m.transactionCategoryIDInput.SetValue("")
					m.transactionInput.Focus()
				case key.Matches(msg, keys.del):
					m.transactionMode = deleteTransactionMode
					m.transactionMessage = ""
					m.transactionIDInput.SetValue("")
					m.transactionIDInput.Focus()
				case msg.String() == "ctrl+f":
					m.transactionMode = filterTransactionMode
					m.transactionMessage = ""
					m.transactionNameFilter.Focus()
					m.transactionDateFrom.Blur()
					m.transactionDateTo.Blur()
				case key.Matches(msg, keys.refresh):
					m.loadTransactions()
				case key.Matches(msg, keys.help):
					m.showHelp = !m.showHelp
				default:
					// Table navigation
					m.transactionTable, cmd = m.transactionTable.Update(msg)
					cmds = append(cmds, cmd)
				}
			case filterTransactionMode:
				inputs := []*textinput.Model{&m.transactionNameFilter, &m.transactionDateFrom, &m.transactionDateTo}
				anyFocused := false
				for _, inp := range inputs {
					if inp.Focused() {
						anyFocused = true
						break
					}
				}
				if anyFocused {
					if key.Matches(msg, keys.tab) {
						focused := -1
						for i, inp := range inputs {
							if inp.Focused() {
								focused = i
								inp.Blur()
								break
							}
						}
						next := (focused + 1) % len(inputs)
						inputs[next].Focus()
					} else if key.Matches(msg, keys.enter) {
						name := m.transactionNameFilter.Value()
						from := m.transactionDateFrom.Value()
						to := m.transactionDateTo.Value()
						if name != "" {
							m.filterTransactionsByName()
						} else if from != "" || to != "" {
							m.filterTransactionsByDate()
						} else {
							m.filteredTransactions = m.transactions
							m.updateTransactionTable()
						}
						m.transactionMode = viewTransactionsMode
						for _, inp := range inputs {
							inp.Blur()
						}
					} else if key.Matches(msg, keys.back) {
						// If esc/back pressed and any input is focused, blur all inputs (do not go back)
						for _, inp := range inputs {
							if inp.Focused() {
								inp.Blur()
							}
						}
						// Do not go back, just blur
						return m, nil
					}
					// Always update the focused input (backspace, typing, etc.)
					var c tea.Cmd
					if m.transactionNameFilter.Focused() {
						m.transactionNameFilter, c = m.transactionNameFilter.Update(msg)
						cmds = append(cmds, c)
					}
					if m.transactionDateFrom.Focused() {
						m.transactionDateFrom, c = m.transactionDateFrom.Update(msg)
						cmds = append(cmds, c)
					}
					if m.transactionDateTo.Focused() {
						m.transactionDateTo, c = m.transactionDateTo.Update(msg)
						cmds = append(cmds, c)
					}
				} else {
					// No input focused: allow esc to go back
					if key.Matches(msg, keys.back) {
						m.transactionMode = viewTransactionsMode
						m.transactionNameFilter.Blur()
						m.transactionDateFrom.Blur()
						m.transactionDateTo.Blur()
					}
				}
				// If no input focused, focus first
				if !anyFocused {
					m.transactionNameFilter.Focus()
					m.transactionDateFrom.Blur()
					m.transactionDateTo.Blur()
				}
			case addTransactionMode:
				inputs := []*textinput.Model{&m.transactionInput, &m.transactionCostInput, &m.transactionDateInput, &m.transactionCategoryIDInput}
				anyFocused := false
				for _, inp := range inputs {
					if inp.Focused() {
						anyFocused = true
						break
					}
				}
				if anyFocused {
					// Update the focused input first
					for _, inp := range inputs {
						if inp.Focused() {
							newModel, cmd := inp.Update(msg)
							*inp = newModel // Important: update the input model with the new state
							cmds = append(cmds, cmd)
							break
						}
					}

					// Handle navigation after input update
					if key.Matches(msg, keys.tab) {
						focused := -1
						for i, inp := range inputs {
							if inp.Focused() {
								focused = i
								inp.Blur()
								break
							}
						}
						next := (focused + 1) % len(inputs)
						inputs[next].Focus()
					} else if key.Matches(msg, keys.enter) {
						name := m.transactionInput.Value()
						costStr := m.transactionCostInput.Value()
						date := strings.TrimSpace(m.transactionDateInput.Value())
						categoryIDStr := m.transactionCategoryIDInput.Value()
						if name == "" || costStr == "" || date == "" || categoryIDStr == "" {
							return m, nil
						}
						cost, err := strconv.ParseFloat(costStr, 64)
						if err != nil {
							return m, nil
						}
						dt, err := time.Parse("2006-01-02", date)
						if err != nil {
							return m, nil
						}
						categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
						if err != nil {
							return m, nil
						}
						if err := m.addTransaction(name, cost, dt.Format("2006-01-02"), categoryID); err == nil {
							m.transactionMode = viewTransactionsMode
							m.transactionInput.SetValue("")
							m.transactionCostInput.SetValue("")
							m.transactionDateInput.SetValue("")
							m.transactionCategoryIDInput.SetValue("")
							m.loadTransactions()
						}
					} else if key.Matches(msg, keys.back) {
						// Just blur all inputs but stay in add mode
						for _, inp := range inputs {
							inp.Blur()
						}
						return m, nil
					}
				} else {
					// No inputs focused - allow esc to exit add mode
					if key.Matches(msg, keys.back) {
						m.transactionMode = viewTransactionsMode
						for _, inp := range inputs {
							inp.Blur()
						}
						return m, nil
					}
					// If no input is focused and not exiting, focus the first input
					m.transactionInput.Focus()
					m.transactionCostInput.Blur()
					m.transactionDateInput.Blur()
					m.transactionCategoryIDInput.Blur()
				}
			case deleteTransactionMode:
				switch {
				case key.Matches(msg, keys.quit):
					return m, tea.Quit
				case key.Matches(msg, keys.back):
					m.transactionMode = viewTransactionsMode
					m.transactionIDInput.Blur()
					m.transactionMessage = ""
				case key.Matches(msg, keys.enter):
					idStr := m.transactionIDInput.Value()
					if idStr == "" {
						m.transactionMessage = "Transaction ID is required"
						break
					}
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err != nil {
						m.transactionMessage = "Invalid transaction ID"
						break
					}
					err = m.deleteTransaction(id)
					if err != nil {
						m.transactionMessage = fmt.Sprintf("Error: %v", err)
					} else {
						m.transactionMessage = "Transaction deleted successfully!"
						m.transactionIDInput.SetValue("")
						m.transactionMode = viewTransactionsMode
						m.loadTransactions()
					}
				default:
					// Always update input for any other key press
					newModel, cmd := m.transactionIDInput.Update(msg)
					m.transactionIDInput = newModel
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

		// Update table size
		m.categoryTable.SetWidth(msg.Width - 4)
		m.categoryTable.SetHeight(msg.Height - 10)

	case tickMsg:
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

func (m *model) loadCategories() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/category", nil)
	if err != nil {
		m.categoryMessage = fmt.Sprintf("Error creating request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+m.authToken)

	resp, err := client.Do(req)
	if err != nil {
		m.categoryMessage = fmt.Sprintf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.categoryMessage = fmt.Sprintf("Error: Status %d", resp.StatusCode)
		return
	}

	var categories []category
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		m.categoryMessage = fmt.Sprintf("Error decoding response: %v", err)
		return
	}

	m.categories = categories
	m.updateCategoryTable()
}

func (m *model) addCategory(name string) error {
	categoryReq := category{Name: name}
	jsonData, err := json.Marshal(categoryReq)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8080/category", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+m.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create category with status: %d", resp.StatusCode)
	}

	return nil
}

func (m *model) deleteCategory(id int64) error {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:8080/category?id=%d", id), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+m.authToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete category with status: %d", resp.StatusCode)
	}

	return nil
}

func (m *model) updateCategoryTable() {
	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: 15},
	}

	rows := []table.Row{}
	for _, cat := range m.categories {
		rows = append(rows, table.Row{
			strconv.FormatInt(cat.ID, 10),
			cat.Name,
		})
	}

	m.categoryTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#fc595f")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#fc595f")).
		Background(lipgloss.Color("#252525")).
		Bold(false)
	m.categoryTable.SetStyles(s)
}

func (m *model) loadTransactions() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/transaction", nil)
	if err != nil {
		m.transactionMessage = fmt.Sprintf("Error creating request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+m.authToken)

	resp, err := client.Do(req)
	if err != nil {
		m.transactionMessage = fmt.Sprintf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.transactionMessage = fmt.Sprintf("Error: Status %d", resp.StatusCode)
		return
	}

	var transactions []transaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		m.transactionMessage = fmt.Sprintf("Error decoding response: %v", err)
		return
	}

	m.transactions = transactions
	m.filteredTransactions = transactions
	m.updateTransactionTable()
}

func (m *model) updateTransactionTable() {
	columns := []table.Column{
		{Title: "ID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "Cost", Width: 10},
		{Title: "Date", Width: 15},
		{Title: "CategoryID", Width: 10},
	}
	rows := []table.Row{}
	for _, t := range m.filteredTransactions {
		rows = append(rows, table.Row{
			strconv.FormatInt(t.ID, 10),
			t.Name,
			fmt.Sprintf("%.2f", t.Cost),
			t.Date.Format("2006-01-02"),
			strconv.FormatInt(t.CategoriesID, 10),
		})
	}
	m.transactionTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	// Style similar to categories
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#fc595f")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#fc595f")).
		Background(lipgloss.Color("#252525")).
		Bold(false)
	m.transactionTable.SetStyles(s)
}

func (m *model) filterTransactionsByName() {
	name := m.transactionNameFilter.Value()
	if name == "" {
		m.filteredTransactions = m.transactions
		m.updateTransactionTable()
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/transaction?name="+name, nil)
	if err != nil {
		m.filteredTransactions = nil
		m.updateTransactionTable()
		return
	}

	req.Header.Set("Authorization", "Bearer "+m.authToken)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		m.filteredTransactions = nil
		m.updateTransactionTable()
		return
	}
	defer resp.Body.Close()

	var filtered []transaction
	if err := json.NewDecoder(resp.Body).Decode(&filtered); err == nil {
		m.filteredTransactions = filtered
	} else {
		m.filteredTransactions = nil
	}
	m.updateTransactionTable()
}

func (m *model) filterTransactionsByDate() {
	from := m.transactionDateFrom.Value()
	to := m.transactionDateTo.Value()
	var filtered []transaction
	for _, t := range m.transactions {
		if from != "" {
			fromTime, err := time.Parse("2006-01-02", from)
			if err != nil {
				continue
			}
			if t.Date.Before(fromTime) {
				continue
			}
		}
		if to != "" {
			toTime, err := time.Parse("2006-01-02", to)
			if err != nil {
				continue
			}
			if t.Date.After(toTime) {
				continue
			}
		}
		filtered = append(filtered, t)
	}
	m.filteredTransactions = filtered
	m.updateTransactionTable()
}

// Add a transaction via HTTP POST
func (m *model) addTransaction(name string, cost float64, date string, categoryID int64) error {
	// Parse date and format as RFC3339
	dt, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format: %v", err)
	}
	transactionReq := struct {
		Name         string  `json:"name"`
		Cost         float64 `json:"cost"`
		Date         string  `json:"date"`
		CategoriesID int64   `json:"categoriesid"`
	}{
		Name:         name,
		Cost:         cost,
		Date:         dt.Format(time.RFC3339),
		CategoriesID: categoryID,
	}
	jsonData, err := json.Marshal(transactionReq)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8080/transaction", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create transaction with status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// Delete a transaction via HTTP DELETE
func (m *model) deleteTransaction(id int64) error {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:8080/transaction?id=%d", id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.authToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("transaction not found")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete transaction with status: %d", resp.StatusCode)
	}
	m.loadTransactions()
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
	case categoryScreen:
		return m.categoryView()
	case transactionScreen:
		return m.transactionView()
	default:
		return m.menuView()
	}
}

func (m model) menuView() string {
	var s strings.Builder

	title := titleStyle.Render("QuattriniTrack - Main Menu")
	s.WriteString(title + "\n\n")

	if m.isLoggedIn {
		s.WriteString(successStyle.Render("✓ Logged in") + "\n\n")
	}

	for i, item := range m.menuItems {
		cursor := "  "
		if i == m.selectedItem {
			cursor = "▶ "
			s.WriteString(selectedMenuStyle.Render(cursor + item.title))
		} else {
			s.WriteString(menuItemStyle.Render(cursor + item.title))
		}
		s.WriteString("\n")

		if i == m.selectedItem {
			s.WriteString(menuItemStyle.Render("    " + item.description))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	if m.showHelp {
		s.WriteString("↑/k: move up • ↓/j: move down • enter: select • ?: toggle help\n")
	} else {
		s.WriteString("Press ? for help\n")
	}

	return s.String()
}

func (m model) authView() string {
	var s strings.Builder

	var title string
	if m.authMode == loginMode {
		title = titleStyle.Render("QuattriniTrack - Login")
	} else {
		title = titleStyle.Render("QuattriniTrack - Register")
	}
	s.WriteString(title + "\n\n")

	s.WriteString("Email:\n")
	if m.focusedInput == 0 {
		s.WriteString(inputStyle.Render(m.emailInput.View()))
	} else {
		s.WriteString(noStyle.Render(m.emailInput.View()))
	}
	s.WriteString("\n\n")

	s.WriteString("Password:\n")
	if m.focusedInput == 1 {
		s.WriteString(inputStyle.Render(m.passwordInput.View()))
	} else {
		s.WriteString(noStyle.Render(m.passwordInput.View()))
	}
	s.WriteString("\n\n")

	if m.authMessage != "" {
		if strings.Contains(m.authMessage, "successful") {
			s.WriteString(successStyle.Render(m.authMessage))
		} else {
			s.WriteString(errorStyle.Render(m.authMessage))
		}
		s.WriteString("\n\n")
	}

	s.WriteString("Tab: next field • Enter: submit • Ctrl+R: toggle login/register • Esc: back to menu\n")

	if m.authMode == loginMode {
		s.WriteString("Press 'Ctrl+R' to switch to register mode\n")
	} else {
		s.WriteString("Press 'Ctrl+R' to switch to login mode\n")
	}

	return s.String()
}

func (m model) categoryView() string {
	var s strings.Builder

	switch m.categoryMode {
	case viewCategoriesMode:
		title := titleStyle.Render("QuattriniTrack - Categories")
		s.WriteString(title + "\n\n")

		if len(m.categories) == 0 {
			s.WriteString("No categories found. Press 'a' to add a category.\n")
		} else {
			s.WriteString(tableStyle.Render(m.categoryTable.View()) + "\n")
		}

		if m.categoryMessage != "" {
			if strings.Contains(m.categoryMessage, "successful") {
				s.WriteString(successStyle.Render(m.categoryMessage))
			} else {
				s.WriteString(errorStyle.Render(m.categoryMessage))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		if m.showHelp {
			s.WriteString("↑/k: move up • ↓/j: move down • ctrl+a: add category • ctrl+d: delete category • r: refresh • esc: back to menu •?: toggle help\n")
		} else {
			s.WriteString("ctrl+a: add • ctrl+d: delete • r: refresh • esc: back • ?: help\n")
		}

	case addCategoryMode:
		title := titleStyle.Render("QuattriniTrack - Add Category")
		s.WriteString(title + "\n\n")

		s.WriteString("Category Name:\n")
		s.WriteString(inputStyle.Render(m.categoryInput.View()))
		s.WriteString("\n\n")

		if m.categoryMessage != "" {
			if strings.Contains(m.categoryMessage, "successful") {
				s.WriteString(successStyle.Render(m.categoryMessage))
			} else {
				s.WriteString(errorStyle.Render(m.categoryMessage))
			}
			s.WriteString("\n\n")
		}

		s.WriteString("Enter: submit • Esc: back to categories\n")

	case deleteCategoryMode:
		title := titleStyle.Render("QuattriniTrack - Delete Category")
		s.WriteString(title + "\n\n")

		s.WriteString("Category ID:\n")
		s.WriteString(inputStyle.Render(m.categoryIDInput.View()))
		s.WriteString("\n\n")

		if m.categoryMessage != "" {
			if strings.Contains(m.categoryMessage, "successful") {
				s.WriteString(successStyle.Render(m.categoryMessage))
			} else {
				s.WriteString(errorStyle.Render(m.categoryMessage))
			}
			s.WriteString("\n\n")
		}

		s.WriteString("Enter: submit • Esc: back to categories\n")
	}

	return s.String()
}

func (m model) transactionView() string {
	var s strings.Builder

	switch m.transactionMode {
	case viewTransactionsMode:
		title := titleStyle.Render("QuattriniTrack - Transactions")
		s.WriteString(title + "\n\n")

		// No filter inputs shown in view mode

		if len(m.filteredTransactions) == 0 {
			s.WriteString("No transactions found. Press 'a' to add a transaction.\n")
		} else {
			s.WriteString(tableStyle.Render(m.transactionTable.View()) + "\n")
		}

		if m.transactionMessage != "" {
			if strings.Contains(m.transactionMessage, "successful") {
				s.WriteString(successStyle.Render(m.transactionMessage))
			} else {
				s.WriteString(errorStyle.Render(m.transactionMessage))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		if m.showHelp {
			s.WriteString("↑/k: move up • ↓/j: move down • ctrl+a: add transaction • ctrl+d: delete transaction • ctrl+f: filter • r: refresh • esc: back to menu • ?: toggle help\n")
		} else {
			s.WriteString("ctrl+a: add • ctrl+d: delete • ctrl+f: filter • r: refresh • esc: back • ?: help\n")
		}
	case addTransactionMode:
		title := titleStyle.Render("QuattriniTrack - Add Transaction")
		s.WriteString(title + "\n\n")
		s.WriteString(inputStyle.Render("Name: "+m.transactionInput.View()) + "\n")
		s.WriteString(inputStyle.Render("Cost: "+m.transactionCostInput.View()) + "\n")
		s.WriteString(inputStyle.Render("Date (YYYY-MM-DD): "+m.transactionDateInput.View()) + "\n")
		s.WriteString(inputStyle.Render("Category ID: "+m.transactionCategoryIDInput.View()) + "\n\n")
		if m.transactionMessage != "" {
			if strings.Contains(m.transactionMessage, "successful") {
				s.WriteString(successStyle.Render(m.transactionMessage))
			} else {
				s.WriteString(errorStyle.Render(m.transactionMessage))
			}
			s.WriteString("\n")
		}
		s.WriteString("Tab: next field • Enter: submit • Esc: back to transactions\n")
	case deleteTransactionMode:
		title := titleStyle.Render("QuattriniTrack - Delete Transaction")
		s.WriteString(title + "\n\n")
		s.WriteString("Transaction ID:\n")
		s.WriteString(inputStyle.Render(m.transactionIDInput.View()))
		s.WriteString("\n\n")
		if m.transactionMessage != "" {
			if strings.Contains(m.transactionMessage, "successful") {
				s.WriteString(successStyle.Render(m.transactionMessage))
			} else {
				s.WriteString(errorStyle.Render(m.transactionMessage))
			}
			s.WriteString("\n\n")
		}
		s.WriteString("Enter: submit • Esc: back to transactions\n")
	case filterTransactionMode:
		title := titleStyle.Render("QuattriniTrack - Filter Transactions")
		s.WriteString(title + "\n\n")
		s.WriteString(inputStyle.Render("Name: "+m.transactionNameFilter.View()) + "\n")
		s.WriteString(inputStyle.Render("Date From: "+m.transactionDateFrom.View()) + "\n")
		s.WriteString(inputStyle.Render("Date To: "+m.transactionDateTo.View()) + "\n\n")
		if m.transactionMessage != "" {
			if strings.Contains(m.transactionMessage, "successful") {
				s.WriteString(successStyle.Render(m.transactionMessage))
			} else {
				s.WriteString(errorStyle.Render(m.transactionMessage))
			}
			s.WriteString("\n")
		}
		s.WriteString("Tab: next field • Enter: apply filter • Esc: cancel\n")
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
		helpText = "\n" + "↑/k: scroll up • ↓/j: scroll down • c: clear logs • esc: back to menu • ?: toggle help"
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

		message := strings.TrimSpace(log.Message)

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
	logger.SetSuppress(true)

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

	categoryInput := textinput.New()
	categoryInput.Placeholder = "Enter category name"
	categoryInput.CharLimit = 50
	categoryInput.Width = 30

	categoryIDInput := textinput.New()
	categoryIDInput.Placeholder = "Enter category ID"
	categoryIDInput.CharLimit = 10
	categoryIDInput.Width = 30

	// Initialize transaction filter inputs FIRST
	transactionNameFilter := textinput.New()
	transactionNameFilter.Placeholder = "Filter by name"
	transactionNameFilter.CharLimit = 50
	transactionNameFilter.Width = 30

	transactionDateFrom := textinput.New()
	transactionDateFrom.Placeholder = "From date (YYYY-MM-DD)"
	transactionDateFrom.CharLimit = 10
	transactionDateFrom.Width = 30

	transactionDateTo := textinput.New()
	transactionDateTo.Placeholder = "To date (YYYY-MM-DD)"
	transactionDateTo.CharLimit = 10
	transactionDateTo.Width = 30

	transactionNameFilter.Focus() // Set initial focus to name filter

	// Initialize transaction inputs
	transactionInput := textinput.New()
	transactionInput.Placeholder = "Enter transaction name"
	transactionInput.CharLimit = 50
	transactionInput.Width = 30

	transactionIDInput := textinput.New()
	transactionIDInput.Placeholder = "Enter transaction ID"
	transactionIDInput.CharLimit = 10
	transactionIDInput.Width = 30

	transactionCostInput := textinput.New()
	transactionCostInput.Placeholder = "Enter cost"
	transactionCostInput.CharLimit = 20
	transactionCostInput.Width = 30

	transactionDateInput := textinput.New()
	transactionDateInput.Placeholder = "Enter date (YYYY-MM-DD)"
	transactionDateInput.CharLimit = 10
	transactionDateInput.Width = 30

	transactionCategoryIDInput := textinput.New()
	transactionCategoryIDInput.Placeholder = "Enter category ID"
	transactionCategoryIDInput.CharLimit = 10
	transactionCategoryIDInput.Width = 30

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
			title:       "Categories",
			description: "Manage categories (view, add, delete) - requires login",
		},
		{
			title:       "Transactions",
			description: "Manage transactions (view, add, delete, filter) - requires login",
		},
		{
			title:       "Exit",
			description: "Close the application",
		},
	}

	// Initialize empty table
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Name", Width: 30},
	}

	categoryTable := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Initialize transaction table
	transactionColumns := []table.Column{
		{Title: "ID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "Cost", Width: 10},
		{Title: "Date", Width: 15},
		{Title: "CategoryID", Width: 10},
	}

	transactionTable := table.New(
		table.WithColumns(transactionColumns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	transactionNameFilter.Focus() // Set initial focus to name filter

	p := tea.NewProgram(
		model{
			lastUpdate:                 time.Now(),
			currentScreen:              menuScreen,
			menuItems:                  menuItems,
			selectedItem:               0,
			emailInput:                 emailInput,
			passwordInput:              passwordInput,
			focusedInput:               0,
			authMode:                   loginMode,
			categoryInput:              categoryInput,
			categoryIDInput:            categoryIDInput,
			categoryTable:              categoryTable,
			categoryMode:               viewCategoriesMode,
			transactionInput:           transactionInput,
			transactionIDInput:         transactionIDInput,
			transactionCostInput:       transactionCostInput,
			transactionDateInput:       transactionDateInput,
			transactionCategoryIDInput: transactionCategoryIDInput,
			transactionTable:           transactionTable,
			transactionMode:            viewTransactionsMode,
			transactionNameFilter:      transactionNameFilter,
			transactionDateFrom:        transactionDateFrom,
			transactionDateTo:          transactionDateTo,
			focusedTransactionInput:    0,
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
	}

	logger.RestoreOriginalOutput()
}
