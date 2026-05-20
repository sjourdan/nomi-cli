package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testNomis() []Nomi {
	return []Nomi{
		{UUID: "u-alice", Name: "Alice", RelationshipType: "Friend"},
		{UUID: "u-bob", Name: "Bob", RelationshipType: "Mentor"},
		{UUID: "u-carol", Name: "Carol", RelationshipType: "Friend"},
	}
}

// sized returns a model that has received a window size, as it would at runtime.
func sized(m chatTUI) chatTUI {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return updated.(chatTUI)
}

func TestSelectNomiCreatesConversation(t *testing.T) {
	m := newChatTUI(testNomis())
	m.selectNomi("u-bob")

	if m.active != "u-bob" {
		t.Fatalf("expected active u-bob, got %q", m.active)
	}
	conv, ok := m.convos["u-bob"]
	if !ok {
		t.Fatal("expected conversation for u-bob to be created")
	}
	if conv.nomi.Name != "Bob" {
		t.Errorf("expected conversation Nomi Bob, got %q", conv.nomi.Name)
	}
}

func TestSwitcherOpenClose(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-bob")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(chatTUI)
	if !m.switcherOpen {
		t.Fatal("Tab should open the switcher")
	}
	// Cursor should land on the active Nomi (Bob, index 1).
	if m.switcherCur != 1 {
		t.Errorf("expected switcher cursor on active Nomi (1), got %d", m.switcherCur)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(chatTUI)
	if m.switcherOpen {
		t.Fatal("Esc should close the switcher")
	}
}

func TestSwitcherNavigateAndSelect(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-alice")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(chatTUI)

	// Move down to Carol (index 0 -> 2) and select.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(chatTUI)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(chatTUI)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(chatTUI)

	if m.switcherOpen {
		t.Error("selecting a Nomi should close the switcher")
	}
	if m.active != "u-carol" {
		t.Errorf("expected active u-carol after selection, got %q", m.active)
	}
}

func TestSwitcherFilter(t *testing.T) {
	m := newChatTUI(testNomis())
	m.filter.SetValue("bo")

	filtered := m.filteredNomis()
	if len(filtered) != 1 || filtered[0].Name != "Bob" {
		t.Errorf("expected filter 'bo' to match only Bob, got %+v", filtered)
	}

	m.filter.SetValue("")
	if len(m.filteredNomis()) != 3 {
		t.Errorf("expected empty filter to match all 3 Nomis")
	}
}

func TestSendAppendsPendingMessage(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-alice")
	m.composer.SetValue("Hello there")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(chatTUI)

	conv := m.convos["u-alice"]
	if len(conv.messages) != 1 || !conv.messages[0].fromUser {
		t.Fatalf("expected one user message after send, got %+v", conv.messages)
	}
	if conv.messages[0].text != "Hello there" {
		t.Errorf("expected sent text preserved, got %q", conv.messages[0].text)
	}
	if !conv.pending {
		t.Error("conversation should be pending a reply after send")
	}
	if cmd == nil {
		t.Error("send should return a command to perform the API call")
	}
	if strings.TrimSpace(m.composer.Value()) != "" {
		t.Error("composer should be cleared after send")
	}
}

func TestEmptySendIsIgnored(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-alice")
	m.composer.SetValue("   ")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(chatTUI)

	if len(m.convos["u-alice"].messages) != 0 {
		t.Error("whitespace-only input should not create a message")
	}
}

func TestReplyRoutedToCorrectConversation(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-alice")
	m.convos["u-alice"].pending = true
	// User switches to Bob while Alice's reply is still in flight.
	m.selectNomi("u-bob")

	updated, _ := m.Update(replyMsg{nomiID: "u-alice", reply: "Hi from Alice"})
	m = updated.(chatTUI)

	alice := m.convos["u-alice"]
	if alice.pending {
		t.Error("Alice's conversation should no longer be pending")
	}
	if len(alice.messages) != 1 || alice.messages[0].text != "Hi from Alice" {
		t.Errorf("reply should land in Alice's conversation, got %+v", alice.messages)
	}
}

func TestReplyErrorRendersErrorMessage(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-alice")
	m.convos["u-alice"].pending = true

	updated, _ := m.Update(replyMsg{nomiID: "u-alice", err: &APIError{StatusCode: 500, Status: "Internal Server Error", Message: "boom"}})
	m = updated.(chatTUI)

	msgs := m.convos["u-alice"].messages
	if len(msgs) != 1 || !msgs[0].isError {
		t.Fatalf("expected one error message, got %+v", msgs)
	}
}

func TestViewDoesNotPanic(t *testing.T) {
	m := sized(newChatTUI(testNomis()))
	m.selectNomi("u-alice")

	if out := m.View(); out == "" {
		t.Error("chat view should render content")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(chatTUI)
	if out := m.View(); !strings.Contains(out, "Switch Nomi") {
		t.Error("switcher view should render the modal title")
	}
}

func TestRunTUIUnknownNomi(t *testing.T) {
	err := runTUI(testNomis(), "Nonexistent")
	if err == nil || !strings.Contains(err.Error(), "no Nomi found") {
		t.Errorf("expected 'no Nomi found' error, got %v", err)
	}
}
