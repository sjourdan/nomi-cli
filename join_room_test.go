package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestJoinRoomLoadsRoom(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	room := Room{UUID: roomID, Name: "Empty room"}
	for _, tt := range []struct {
		name     string
		selector string
		lookup   bool
	}{
		{name: "by UUID", selector: roomID},
		{name: "by name", selector: "empty ROOM", lookup: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case tt.lookup && r.Method == http.MethodGet && r.URL.Path == "/v1/rooms":
					json.NewEncoder(w).Encode(RoomResponse{Rooms: []Room{room}})
				case r.Method == http.MethodGet && r.URL.Path == "/v1/rooms/"+roomID:
					json.NewEncoder(w).Encode(room)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			previousKey, previousURL, previousClient := apiKey, baseURL, client
			t.Cleanup(func() { apiKey, baseURL, client = previousKey, previousURL, previousClient })
			t.Setenv("NOMI_API_URL", server.URL+"/v1")

			cmd := newRootCmd()
			cmd.SetArgs([]string{"join-room", tt.selector, "--api-key", "test-api-key"})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), `room "Empty room" has no Nomis`) {
				t.Fatalf("expected a message about the loaded room, got %v", err)
			}
		})
	}
}

func TestRoomChatRunsFiveRepliesAfterOneUserMessage(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	const text = "What's everyone reading?"
	room := Room{UUID: roomID, Name: "Book club", Nomis: testNomis()}
	speakers := map[string]string{"u-alice": "Alice", "u-bob": "Bob", "u-carol": "Carol"}
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
		call := calls.Add(1)
		switch call {
		case 1:
			if r.URL.Path != path || body["messageText"] != text {
				t.Errorf("expected user message first, got %s %+v", r.URL.Path, body)
			}
			json.NewEncoder(w).Encode(map[string]Message{"sentMessage": {UUID: "sent-id", Text: text, Sent: "2026-11-25T12:00:00Z"}})
		case 2, 3, 4, 5, 6:
			id := body["nomiUuid"]
			speaker := speakers[id]
			if r.URL.Path != path+"/request" || speaker == "" {
				t.Errorf("expected room reply request, got %s %+v", r.URL.Path, body)
			}
			if call == 2 && speaker != "Bob" {
				t.Errorf("expected Bob to start the turn, got %q", speaker)
			}
			json.NewEncoder(w).Encode(map[string]Message{"replyMessage": {UUID: "reply-id", Text: "Reply from " + speaker, Sent: "2026-11-25T12:00:01Z"}})
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
	cmd = commands[0]
	for i := 1; i <= 5; i++ {
		if cmd == nil {
			t.Fatalf("room stopped before reply %d", i)
		}
		reply, ok := cmd().(replyMsg)
		if !ok || reply.err != nil {
			t.Fatalf("expected room reply %d, got %+v", i, reply)
		}
		updated, cmd = updated.Update(reply)
		if i == 1 {
			m = updated.(chatTUI)
			m.composer.SetValue("My next turn")
			var earlySend tea.Cmd
			updated, earlySend = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if earlySend != nil || updated.(chatTUI).composer.Value() != "My next turn" {
				t.Fatal("Enter should keep the next draft until the room turn ends")
			}
		}
	}
	m = updated.(chatTUI)
	if cmd != nil || m.anyPending() {
		t.Error("room should wait for a human turn after five replies")
	}
	messages := m.convos[roomID].messages
	if len(messages) != 6 || !messages[0].fromUser || messages[0].text != text || messages[1].speaker != "Bob" {
		t.Fatalf("expected one user message followed by five replies starting with Bob, got %+v", messages)
	}
	for i, msg := range messages[1:] {
		if msg.text != "Reply from "+msg.speaker || msg.speaker == "" {
			t.Errorf("reply %d was not attributed to its speaker: %+v", i+1, msg)
		}
		if msg.speaker == messages[i].speaker {
			t.Errorf("%s spoke twice in a row", msg.speaker)
		}
	}
	if got := calls.Load(); got != 6 {
		t.Errorf("issued %d API calls, want 6", got)
	}
}

func TestRoomChatCanEndBeforeFiveReplies(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	nomis := testNomis()
	for _, tt := range []struct {
		name  string
		nomis []Nomi
		err   error
	}{
		{name: "only one participant", nomis: nomis[:1]},
		{name: "reply request failed", nomis: nomis, err: errors.New("no reply")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := sized(newRoomChatTUI(Room{UUID: roomID, Name: "Book club", Nomis: tt.nomis}))
			m.composer.SetValue("Hello")
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			updated, next := updated.Update(replyMsg{
				conversationID: roomID, speaker: tt.nomis[0], reply: "Hi", roomRepliesRemaining: 4, err: tt.err,
			})
			m = updated.(chatTUI)
			if next != nil || m.anyPending() {
				t.Error("room should wait for the next human turn")
			}
			messages := m.convos[roomID].messages
			if len(messages) != 2 || messages[1].isError != (tt.err != nil) {
				t.Errorf("unexpected room transcript: %+v", messages)
			}
		})
	}
}
