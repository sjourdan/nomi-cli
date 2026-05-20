package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFetchNomis tests the function for fetching all Nomis
func TestFetchNomis(t *testing.T) {
	// Setup a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method and path
		if r.Method != "GET" || r.URL.Path != "/nomis" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check for authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"nomis": [
				{
					"uuid": "123",
					"name": "Alice",
					"gender": "Female",
					"created": "2023-01-01",
					"relationshipType": "Friend"
				},
				{
					"uuid": "456",
					"name": "Bob",
					"gender": "Male",
					"created": "2023-01-02",
					"relationshipType": "Mentor"
				}
			]
		}`))
	}))
	defer mockServer.Close()

	// Set the API key and URL
	apiKey = "test-api-key"
	baseURL = mockServer.URL

	// Initialize the client for testing
	client = NewNomiClient(apiKey, baseURL)

	// Test fetching Nomis
	nomis, err := client.GetNomis()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Check the number of Nomis
	if len(nomis) != 2 {
		t.Errorf("Expected 2 Nomis, got %d", len(nomis))
		return
	}

	// Check the first Nomi
	if nomis[0].UUID != "123" || nomis[0].Name != "Alice" || nomis[0].RelationshipType != "Friend" {
		t.Errorf("First Nomi doesn't match expected values: %+v", nomis[0])
	}

	// Check the second Nomi
	if nomis[1].UUID != "456" || nomis[1].Name != "Bob" || nomis[1].RelationshipType != "Mentor" {
		t.Errorf("Second Nomi doesn't match expected values: %+v", nomis[1])
	}
}
