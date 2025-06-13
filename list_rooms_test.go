package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDisplayRoom(t *testing.T) {
	tests := []struct {
		name   string
		room   Room
		output []string
	}{
		{
			name: "Empty name",
			room: Room{
				UUID:    "123",
				Created: "2024-12-07T10:00:00Z",
				Updated: "2024-12-07T11:00:00Z",
				Status:  "active",
			},
			output: []string{
				"Room: <empty>",
				"- UUID: 123",
				"- Created: 2024-12-07T10:00:00Z",
				"- Updated: 2024-12-07T11:00:00Z",
				"- Status: active",
				"- Backchanneling: false",
			},
		},
		{
			name: "With name, note and nomis",
			room: Room{
				Name:                  "Test Room",
				UUID:                  "abc-123",
				Created:               "2024-12-07T10:00:00Z",
				Updated:               "2024-12-07T11:00:00Z",
				Status:                "closed",
				BackchannelingEnabled: true,
				Note:                  "This is a note",
				Nomis: []Nomi{
					{
						Name:             "Nomi1",
						Gender:           "female",
						RelationshipType: "friend",
					},
					{
						Name:             "Nomi2",
						Gender:           "male",
						RelationshipType: "colleague",
					},
				},
			},
			output: []string{
				"Room: Test Room",
				"- UUID: abc-123",
				"- Created: 2024-12-07T10:00:00Z",
				"- Updated: 2024-12-07T11:00:00Z",
				"- Status: closed",
				"- Backchanneling: true",
				"- Note: This is a note",
				"- Nomis:",
				"• Nomi1 (female, friend)",
				"• Nomi2 (male, colleague)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			displayRoom(tt.room)

			w.Close()
			os.Stdout = old
			out, _ := io.ReadAll(r)
			outputStr := string(out)

			for _, line := range tt.output {
				if !strings.Contains(outputStr, line) {
					t.Errorf("Expected line %q in output, got: %q", line, outputStr)
				}
			}
		})
	}
}

func TestListRoomsCmd(t *testing.T) {
	// Create a test server to mock the /rooms endpoint
	rooms := []Room{
		{
			Name:    "Room1",
			UUID:    "uuid-room1",
			Created: "2024-12-07T09:00:00Z",
			Updated: "2024-12-07T10:00:00Z",
			Status:  "active",
		},
		{
			Name:                  "Room2",
			UUID:                  "uuid-room2",
			Created:               "2024-12-06T09:00:00Z",
			Updated:               "2024-12-06T10:00:00Z",
			Status:                "inactive",
			BackchannelingEnabled: true,
			Note:                  "Some note",
			Nomis: []Nomi{
				{
					Name:             "Alice",
					Gender:           "female",
					RelationshipType: "friend",
				},
			},
		},
	}

	testResp := RoomResponse{
		Rooms: rooms,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header if desired
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + apiKey
		if authHeader != expectedAuth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/rooms":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(testResp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Override baseURL and initialize client for testing
	baseURL = server.URL
	apiKey = "test-api-key"
	client = NewNomiClient(apiKey, baseURL)

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(listRoomsCmd)

	// Since listRoomsCmd has no arguments, just run it
	cmd.SetArgs([]string{"list-rooms"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	outputStr := string(out)

	// Check that the output contains expected lines
	expectedLines := []string{
		"Total Rooms: 2",
		"Room: Room1",
		"Room: Room2",
		"- Nomis:",
		"  • Alice (female, friend)",
	}

	for _, line := range expectedLines {
		if !strings.Contains(outputStr, line) {
			t.Errorf("Expected output to contain %q, but got: %q", line, outputStr)
		}
	}
}

func TestListRoomsCmdError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 500 to simulate server error
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override baseURL and initialize client for testing
	baseURL = server.URL
	apiKey = "test-api-key"
	client = NewNomiClient(apiKey, baseURL)

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(listRoomsCmd)

	cmd.SetArgs([]string{"list-rooms"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	outputStr := string(out)

	// We expect an error message in the output
	if !strings.Contains(outputStr, "API error (500") {
		t.Errorf("Expected output to contain API error 500, got %q", outputStr)
	}
}
