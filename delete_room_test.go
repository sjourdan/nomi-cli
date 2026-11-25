package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDeleteRoomCommand(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	room := Room{UUID: roomID, Name: "Book club"}
	otherRoom := Room{UUID: "f4d1dacc-0081-4894-a7cc-3fc4179d9121", Name: "BOOK CLUB"}
	for _, tt := range []struct {
		name     string
		selector string
		rooms    []Room
		wantErr  string
	}{
		{name: "delete by name", selector: "book club", rooms: []Room{room}},
		{name: "delete by UUID with duplicate names", selector: roomID, rooms: []Room{otherRoom, room}},
		{name: "reject ambiguous name", selector: room.Name, rooms: []Room{room, otherRoom}, wantErr: "multiple rooms named"},
		{name: "reject empty selector", rooms: []Room{{UUID: roomID}}, wantErr: "provide a room name or UUID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var deletes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/v1/rooms":
					json.NewEncoder(w).Encode(RoomResponse{Rooms: tt.rooms})
				case r.Method == http.MethodDelete && r.URL.Path == "/v1/rooms/"+roomID:
					deletes.Add(1)
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			previousKey, previousURL, previousClient := apiKey, baseURL, client
			t.Cleanup(func() { apiKey, baseURL, client = previousKey, previousURL, previousClient })
			t.Setenv("NOMI_API_KEY", "test-api-key")
			t.Setenv("NOMI_API_URL", server.URL+"/v1")

			var output bytes.Buffer
			cmd := newRootCmd()
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"delete-room", tt.selector})
			err := cmd.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				if deletes.Load() != 0 {
					t.Error("invalid room selector sent a delete request")
				}
				return
			}
			if err != nil {
				t.Fatalf("delete-room failed: %v", err)
			}
			if deletes.Load() != 1 || !strings.Contains(output.String(), "Deleted room") {
				t.Errorf("expected one delete and a confirmation, got %d deletes and %q", deletes.Load(), output.String())
			}
		})
	}
}
