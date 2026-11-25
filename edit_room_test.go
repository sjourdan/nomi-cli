package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestEditRoomCommand(t *testing.T) {
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	const courtneyID = "f4d1dacc-0081-4894-a7cc-3fc4179d9121"
	const aliceID = "81837c4d-0282-48a7-8a36-4c25ac8c6cc1"
	nomis := []Nomi{{UUID: courtneyID, Name: "Courtney Love"}, {UUID: aliceID, Name: "Alice"}}
	for _, tt := range []struct {
		name           string
		room           string
		flags          []string
		body           map[string]any
		readback       []Nomi
		wantName       string
		wantErr        string
		wantAbsentNomi string
	}{
		{
			name: "edit all fields by room name",
			room: "book club",
			flags: []string{"--name", "Weekend readers", "--note", "Bring a book", "--backchanneling=false",
				"--nomi", "Alice", "--nomi", "Courtney Love"},
			body: map[string]any{
				"name": "Weekend readers", "note": "Bring a book", "backchannelingEnabled": false,
				"nomiUuids": []any{aliceID, courtneyID},
			},
			wantName: "Weekend readers",
		},
		{
			name:     "clear just the note by UUID",
			room:     roomID,
			flags:    []string{"--note", ""},
			body:     map[string]any{"note": ""},
			wantName: "Book club",
		},
		{
			name:     "report when the API retains an omitted participant",
			room:     "book club",
			flags:    []string{"--nomi", "Alice"},
			body:     map[string]any{"nomiUuids": []any{aliceID}},
			readback: nomis,
			wantName: "Book club",
			wantErr:  "did not apply the requested participant list",
		},
		{
			name:           "show the confirmed participants when the update response is stale",
			room:           "book club",
			flags:          []string{"--nomi", "Alice"},
			body:           map[string]any{"nomiUuids": []any{aliceID}},
			readback:       nomis[1:],
			wantName:       "Book club",
			wantAbsentNomi: "Courtney Love",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/v1/rooms":
					json.NewEncoder(w).Encode(RoomResponse{Rooms: []Room{{UUID: roomID, Name: "Book club"}}})
				case r.Method == http.MethodGet && r.URL.Path == "/v1/nomis" && tt.body["nomiUuids"] != nil:
					json.NewEncoder(w).Encode(NomiResponse{Nomis: nomis})
				case r.Method == http.MethodGet && r.URL.Path == "/v1/rooms/"+roomID && tt.readback != nil:
					json.NewEncoder(w).Encode(Room{UUID: roomID, Name: tt.wantName, Status: "Default", Nomis: tt.readback})
				case r.Method == http.MethodPut && r.URL.Path == "/v1/rooms/"+roomID:
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if !reflect.DeepEqual(body, tt.body) {
						t.Errorf("update body = %#v, want %#v", body, tt.body)
					}
					json.NewEncoder(w).Encode(Room{UUID: roomID, Name: tt.wantName, Status: "Default", Nomis: nomis})
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

			var output, errorOutput bytes.Buffer
			cmd := newRootCmd()
			cmd.SetOut(&output)
			cmd.SetErr(&errorOutput)
			cmd.SetArgs(append([]string{"edit-room", tt.room}, tt.flags...))
			err := cmd.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || !strings.Contains(err.Error(), "Courtney Love") {
					t.Fatalf("expected a participant mismatch identifying Courtney Love, got %v", err)
				}
				if output.Len() != 0 {
					t.Errorf("mismatched update wrote to stdout: %q", output.String())
				}
				if got := errorOutput.String(); strings.Count(got, tt.wantErr) != 1 || strings.Contains(got, "Usage:") {
					t.Errorf("expected a single error with no usage text on stderr, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("edit-room failed: %v", err)
			}
			if !strings.Contains(output.String(), "Room: "+tt.wantName) {
				t.Errorf("output should show the updated room, got %q", output.String())
			}
			if tt.wantAbsentNomi != "" && strings.Contains(output.String(), tt.wantAbsentNomi) {
				t.Errorf("output should use the confirmed participant list, got %q", output.String())
			}
		})
	}
}
