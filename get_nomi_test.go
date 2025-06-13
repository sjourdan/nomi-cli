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

func TestGetNomiCmdSuccess(t *testing.T) {
	// Create a test Nomi
	testNomi := Nomi{
		UUID:             "nomi-123",
		Name:             "Test Nomi",
		Gender:           "non-binary",
		Created:          "2024-12-07T10:00:00Z",
		RelationshipType: "friend",
	}

	// Start a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header if needed
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + apiKey
		if authHeader != expectedAuth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/nomis/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(testNomi)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Override baseURL and initialize client
	baseURL = server.URL
	apiKey = "test-api-key"
	client = NewNomiClient(apiKey, baseURL)

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Construct a parent command and add getNomiCmd to it
	rootCmd := &cobra.Command{Use: "test"}
	rootCmd.AddCommand(getNomiCmd)

	// Execute the command with the test ID
	rootCmd.SetArgs([]string{"get-nomi", "nomi-123"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	wOut.Close()
	os.Stdout = oldStdout
	outBytes, _ := io.ReadAll(rOut)
	outputStr := string(outBytes)

	// Check that output contains expected lines
	expectedLines := []string{
		"Nomi Details:",
		"- ID: nomi-123",
		"- Name: Test Nomi",
		"- Gender: non-binary",
		"- Created: 2024-12-07T10:00:00Z",
		"- Relationship Type: friend",
	}

	for _, line := range expectedLines {
		if !strings.Contains(outputStr, line) {
			t.Errorf("Expected output to contain %q, got %q", line, outputStr)
		}
	}
}

func TestGetNomiCmdNotFound(t *testing.T) {
	// Start a mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Override baseURL and initialize client
	baseURL = server.URL
	apiKey = "test-api-key"
	client = NewNomiClient(apiKey, baseURL)

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rootCmd := &cobra.Command{Use: "test"}
	rootCmd.AddCommand(getNomiCmd)

	// Execute the command with some test ID
	rootCmd.SetArgs([]string{"get-nomi", "invalid-id"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	wOut.Close()
	os.Stdout = oldStdout
	outBytes, _ := io.ReadAll(rOut)
	outputStr := string(outBytes)

	// Check that output indicates an error
	if !strings.Contains(outputStr, "API error (404") {
		t.Errorf("Expected output to contain API error 404, got %q", outputStr)
	}
}

func TestGetNomiCmdServerError(t *testing.T) {
	// Start a mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override baseURL and initialize client
	baseURL = server.URL
	apiKey = "test-api-key"
	client = NewNomiClient(apiKey, baseURL)

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rootCmd := &cobra.Command{Use: "test"}
	rootCmd.AddCommand(getNomiCmd)

	rootCmd.SetArgs([]string{"get-nomi", "some-id"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	wOut.Close()
	os.Stdout = oldStdout
	outBytes, _ := io.ReadAll(rOut)
	outputStr := string(outBytes)

	if !strings.Contains(outputStr, "API error (500") {
		t.Errorf("Expected output to contain API error 500, got %q", outputStr)
	}
}

func TestGetNomiCmdBadArguments(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	rootCmd.AddCommand(getNomiCmd)

	// Since error messages and usage go to stderr by default, we need to capture stderr as well.
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	rootCmd.SetArgs([]string{"get-nomi"}) // no ID provided
	err := rootCmd.Execute()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)

	outputStr := string(outBytes)
	errorStr := string(errBytes)

	if err == nil {
		t.Errorf("Expected an error when no arguments are provided.")
	}

	// Since Cobra usage errors go to stderr, we check errorStr
	if !strings.Contains(errorStr, "accepts 1 arg(s), received 0") {
		t.Errorf("Expected error message about missing argument, got %q (stdout=%q)", errorStr, outputStr)
	}
}
