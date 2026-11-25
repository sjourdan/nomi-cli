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

func TestCreateRoomCommand(t *testing.T) {
	const courtneyID = "f4d1dacc-0081-4894-a7cc-3fc4179d9121"
	const bobID = "5896ee50-482d-47ed-a6f2-f0170ca41202"
	const aliceID = "81837c4d-0282-48a7-8a36-4c25ac8c6cc1"
	const roomID = "5b183c97-31ac-42fb-b35d-d4333c502872"
	nomis := []Nomi{{UUID: courtneyID, Name: "Courtney Love"}, {UUID: bobID, Name: "Bob"}, {UUID: aliceID, Name: "Alice"}}
	var lookups atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/nomis" {
			lookups.Add(1)
			json.NewEncoder(w).Encode(NomiResponse{Nomis: nomis})
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/rooms" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q, want application/json", got)
		}
		var body struct {
			Name         string   `json:"name"`
			Note         string   `json:"note"`
			Backchannel  bool     `json:"backchannelingEnabled"`
			Participants []string `json:"nomiUuids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Name != "Book club" || body.Note != "Bring a book" || !body.Backchannel ||
			strings.Join(body.Participants, ",") != courtneyID+","+bobID+","+aliceID {
			t.Errorf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Room{
			UUID: roomID, Name: "Book club", Status: "Creating", Note: "Bring a book",
			BackchannelingEnabled: true,
			Nomis:                 nomis,
		})
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
	cmd.SetArgs([]string{"create-room", "Book club", "--nomi", "courtney love", "--nomi", "Bob", "--nomi", aliceID, "--note", "Bring a book", "--backchanneling"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("create-room failed: %v", err)
	}
	if !strings.Contains(output.String(), "Room: Book club") || !strings.Contains(output.String(), roomID) {
		t.Errorf("output should identify the created room, got %q", output.String())
	}
	if !strings.Contains(output.String(), "Courtney Love") || strings.Contains(output.String(), courtneyID) {
		t.Errorf("output should identify participants by name, got %q", output.String())
	}
	if got := lookups.Load(); got != 1 {
		t.Errorf("looked up Nomis %d times, want 1", got)
	}
}

func TestCreateRoomRejectsUnresolvedNomiNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/nomis" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(NomiResponse{Nomis: []Nomi{
			{UUID: "f4d1dacc-0081-4894-a7cc-3fc4179d9121", Name: "Courtney Love"},
			{UUID: "5896ee50-482d-47ed-a6f2-f0170ca41202", Name: "Courtney Love"},
		}})
	}))
	defer server.Close()

	previousClient := client
	t.Cleanup(func() { client = previousClient })
	client = NewNomiClient("test-api-key", server.URL+"/v1")
	for _, tt := range []struct {
		name string
		want string
	}{
		{name: "Courtney Love", want: `multiple Nomis named "Courtney Love"`},
		{name: "Missing Nomi", want: `no Nomi matches "Missing Nomi"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCreateRoomCmd()
			cmd.SetArgs([]string{"Book club", "--nomi", tt.name})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want an error containing %q, got %v", tt.want, err)
			}
		})
	}
}
