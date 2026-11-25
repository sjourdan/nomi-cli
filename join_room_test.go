package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestJoinRoomLoadsRoomByUUID(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/rooms/"+roomID {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(Room{UUID: roomID, Name: "Empty room"})
	}))
	defer server.Close()

	previousKey, previousURL, previousClient := apiKey, baseURL, client
	t.Cleanup(func() { apiKey, baseURL, client = previousKey, previousURL, previousClient })
	t.Setenv("NOMI_API_URL", server.URL+"/v1")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"join-room", roomID, "--api-key", "test-api-key"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), `room "Empty room" has no Nomis`) {
		t.Fatalf("expected a message about the loaded room, got %v", err)
	}
}

func TestRoomChatPostsBeforeRequestingSelectedNomi(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	const text = "What's everyone reading?"
	room := Room{UUID: roomID, Name: "Book club", Nomis: testNomis()}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		path := "/v1/rooms/" + roomID + "/chat"
		w.Header().Set("Content-Type", "application/json")
		switch calls.Add(1) {
		case 1:
			if r.URL.Path != path || body["messageText"] != text {
				t.Errorf("expected user message first, got %s %+v", r.URL.Path, body)
			}
			json.NewEncoder(w).Encode(map[string]Message{"sentMessage": {UUID: "sent-id", Text: text, Sent: "2026-11-25T12:00:00Z"}})
		case 2:
			if r.URL.Path != path+"/request" || body["nomiUuid"] != "u-bob" {
				t.Errorf("expected reply request for Bob, got %s %+v", r.URL.Path, body)
			}
			json.NewEncoder(w).Encode(map[string]Message{"replyMessage": {UUID: "reply-id", Text: "A mystery.", Sent: "2026-11-25T12:00:01Z"}})
		default:
			t.Error("unexpected additional request")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	m := sized(newRoomChatTUI(room))
	m.client = NewNomiClient("test-api-key", server.URL+"/v1")
	for _, key := range []tea.KeyType{tea.KeyTab, tea.KeyDown, tea.KeyEnter} {
		updated, _ := m.Update(tea.KeyMsg{Type: key})
		m = updated.(chatTUI)
	}
	if m.active != roomID || m.roomSpeaker.Name != "Bob" {
		t.Fatalf("expected Bob selected in the same room, got %q in %q", m.roomSpeaker.Name, m.active)
	}
	m.composer.SetValue(text)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("room send did not dispatch a command")
	}
	commands, ok := cmd().(tea.BatchMsg)
	if !ok || len(commands) == 0 {
		t.Fatal("expected a room request and typing indicator")
	}
	reply, ok := commands[0]().(replyMsg)
	if !ok || reply.err != nil {
		t.Fatalf("expected a room reply, got %+v", reply)
	}
	updated, _ = updated.Update(reply)
	m = updated.(chatTUI)
	messages := m.convos[roomID].messages
	if len(messages) != 2 || !messages[0].fromUser || messages[0].text != text ||
		messages[1].speaker != "Bob" || messages[1].text != "A mystery." {
		t.Errorf("unexpected room transcript: %+v", messages)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("issued %d API calls, want 2", got)
	}
}
