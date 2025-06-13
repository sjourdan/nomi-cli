package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelInit(t *testing.T) {
	mockNomis := []Nomi{
		{UUID: "123", Name: "Alice", RelationshipType: "Friend"},
		{UUID: "456", Name: "Bob", RelationshipType: "Mentor"},
	}

	m := model{
		nomis:    mockNomis,
		cursor:   0,
		selected: -1,
	}

	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Expected nil command from Init(), got %v", cmd)
	}
}

func TestModelUpdate(t *testing.T) {
	mockNomis := []Nomi{
		{UUID: "123", Name: "Alice", RelationshipType: "Friend"},
		{UUID: "456", Name: "Bob", RelationshipType: "Mentor"},
	}

	tests := []struct {
		name           string
		initialCursor  int
		keyMsg         tea.KeyMsg
		expectedCursor int
		expectedQuit   bool
	}{
		{
			name:           "down key moves cursor down",
			initialCursor:  0,
			keyMsg:         tea.KeyMsg{Type: tea.KeyDown},
			expectedCursor: 1,
			expectedQuit:   false,
		},
		{
			name:           "j key moves cursor down",
			initialCursor:  0,
			keyMsg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			expectedCursor: 1,
			expectedQuit:   false,
		},
		{
			name:           "up key moves cursor up",
			initialCursor:  1,
			keyMsg:         tea.KeyMsg{Type: tea.KeyUp},
			expectedCursor: 0,
			expectedQuit:   false,
		},
		{
			name:           "k key moves cursor up",
			initialCursor:  1,
			keyMsg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}},
			expectedCursor: 0,
			expectedQuit:   false,
		},
		{
			name:           "up key at top boundary doesn't move cursor",
			initialCursor:  0,
			keyMsg:         tea.KeyMsg{Type: tea.KeyUp},
			expectedCursor: 0,
			expectedQuit:   false,
		},
		{
			name:           "down key at bottom boundary doesn't move cursor",
			initialCursor:  1,
			keyMsg:         tea.KeyMsg{Type: tea.KeyDown},
			expectedCursor: 1,
			expectedQuit:   false,
		},
		{
			name:           "enter key selects current item and quits",
			initialCursor:  1,
			keyMsg:         tea.KeyMsg{Type: tea.KeyEnter},
			expectedCursor: 1,
			expectedQuit:   true,
		},
		{
			name:           "q key quits",
			initialCursor:  0,
			keyMsg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectedCursor: 0,
			expectedQuit:   true,
		},
		{
			name:           "ctrl+c quits",
			initialCursor:  0,
			keyMsg:         tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedCursor: 0,
			expectedQuit:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				nomis:    mockNomis,
				cursor:   tt.initialCursor,
				selected: -1,
			}

			updatedModel, cmd := m.Update(tt.keyMsg)

			// Type assertion
			updated, ok := updatedModel.(model)
			if !ok {
				t.Fatalf("Expected model type, got %T", updatedModel)
			}

			// Check cursor position
			if updated.cursor != tt.expectedCursor {
				t.Errorf("Expected cursor to be %d, got %d",
					tt.expectedCursor, updated.cursor)
			}

			// For Enter key, check that selection was made
			if tt.keyMsg.Type == tea.KeyEnter {
				if updated.selected != tt.initialCursor {
					t.Errorf("Expected selection to be %d after Enter, got %d",
						tt.initialCursor, updated.selected)
				}
			}

			// For quit commands, we can only verify that a command was returned
			// Full tea.Quit testing would require a more complex mock setup
			if tt.expectedQuit && cmd == nil {
				t.Errorf("Expected a command to be returned for quit action")
			}
		})
	}
}

func TestModelView(t *testing.T) {
	mockNomis := []Nomi{
		{UUID: "123", Name: "Alice", RelationshipType: "Friend"},
		{UUID: "456", Name: "Bob", RelationshipType: "Mentor"},
	}

	m := model{
		nomis:    mockNomis,
		cursor:   0,
		selected: -1,
	}

	output := m.View()

	// Test that view contains all Nomis
	for _, nomi := range mockNomis {
		if !contains(output, nomi.Name) {
			t.Errorf("View output doesn't contain Nomi name: %s", nomi.Name)
		}

		if !contains(output, nomi.RelationshipType) {
			t.Errorf("View output doesn't contain relationship type: %s", nomi.RelationshipType)
		}
	}

	// Test that view contains the names
	if !contains(output, "Alice") {
		t.Errorf("View output doesn't contain first Nomi name")
	}

	if !contains(output, "Bob") {
		t.Errorf("View output doesn't contain second Nomi name")
	}

	// Move cursor to second item
	m.cursor = 1
	output = m.View()

	// Test that both names are still present
	if !contains(output, "Alice") || !contains(output, "Bob") {
		t.Errorf("View output doesn't contain all Nomi names after cursor update")
	}
}

// Test edge cases for the selectableMenu function
func TestSelectableMenu(t *testing.T) {
	// Test empty list
	_, err := selectableMenu([]Nomi{})
	if err == nil || err.Error() != "no Nomis found" {
		t.Errorf("Expected 'no Nomis found' error, got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
