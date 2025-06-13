package main

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
)

// model represents the UI state of our menu
type model struct {
	nomis    []Nomi
	cursor   int
	selected int
}

// Init initializes the bubbletea model
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles key press events
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.nomis)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the UI
func (m model) View() string {
	// Title with styling
	s := fmt.Sprintf("\n%s=== Select a Nomi to Chat With ===%s\n\n", colorYellow, colorReset)

	// List each Nomi with styling
	for i, nomi := range m.nomis {
		cursor := "  "
		if m.cursor == i {
			// Highlight the selected item
			cursor = fmt.Sprintf("%s>%s ", colorGreen, colorReset)
			s += fmt.Sprintf("%s%s%s (%s%s%s)\n",
				cursor,
				colorGreen, nomi.Name, colorBlue, nomi.RelationshipType, colorReset)
		} else {
			s += fmt.Sprintf("%s%s (%s%s%s)\n",
				cursor,
				nomi.Name, colorCyan, nomi.RelationshipType, colorReset)
		}
	}

	// Instructions with styling
	s += fmt.Sprintf("\n%s• Use arrow keys or j/k to navigate\n", colorBlue)
	s += fmt.Sprintf("• Press Enter to start chat\n")
	s += fmt.Sprintf("• Press q to quit%s\n", colorReset)

	return s
}

// selectableMenu creates and runs a TUI for Nomi selection
func selectableMenu(nomis []Nomi) (Nomi, error) {
	if len(nomis) == 0 {
		return Nomi{}, fmt.Errorf("no Nomis found")
	}

	// Clear the screen before showing the menu
	clearScreen()

	initialModel := model{
		nomis:    nomis,
		cursor:   0,
		selected: -1,
	}

	p := tea.NewProgram(initialModel)
	finalModel, err := p.Run()
	if err != nil {
		return Nomi{}, fmt.Errorf("error running menu: %v", err)
	}

	if m, ok := finalModel.(model); ok {
		if m.selected == -1 {
			return Nomi{}, fmt.Errorf("no Nomi selected")
		}
		return m.nomis[m.selected], nil
	}

	return Nomi{}, fmt.Errorf("unexpected model type")
}
