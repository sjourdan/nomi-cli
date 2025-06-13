package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindNomiByName(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Error("Expected Authorization header with API key")
		}

		// Mock response
		response := struct {
			Nomis []struct {
				UUID string `json:"uuid"`
				Name string `json:"name"`
			} `json:"nomis"`
		}{
			Nomis: []struct {
				UUID string `json:"uuid"`
				Name string `json:"name"`
			}{
				{
					UUID: "test-uuid-1",
					Name: "John",
				},
				{
					UUID: "test-uuid-2",
					Name: "Alice",
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set test environment
	apiKey = "test-api-key"
	baseURL = server.URL

	// Initialize the client for testing
	client = NewNomiClient(apiKey, baseURL)

	// Test finding existing Nomi
	uuid, err := findNomiByName("John")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if uuid != "test-uuid-1" {
		t.Errorf("Expected UUID test-uuid-1, got %s", uuid)
	}

	// Test finding non-existent Nomi
	_, err = findNomiByName("NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent Nomi, got none")
	}
}

func TestChatRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Decode request body
		var chatReq ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
			t.Errorf("Error decoding request body: %v", err)
		}

		// Mock response
		response := ChatResponse{
			SentMessage: Message{
				UUID: "msg-1",
				Text: chatReq.MessageText,
				Sent: "2024-01-01T12:00:00Z",
			},
			ReplyMessage: Message{
				UUID: "msg-2",
				Text: "Test response",
				Sent: "2024-01-01T12:00:01Z",
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set test environment
	baseURL = server.URL
	apiKey = "test-api-key"

	// Test chat request/response
	client := &http.Client{}
	chatReq := ChatRequest{MessageText: "Hello"}
	body, _ := json.Marshal(chatReq)
	req, _ := http.NewRequest("POST", server.URL+"/nomis/test-uuid/chat", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
}
