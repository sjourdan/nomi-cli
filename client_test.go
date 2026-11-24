package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// chatOKResponse writes a valid ChatResponse echoing the incoming message.
func chatOKResponse(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(ChatResponse{
		SentMessage:  Message{UUID: "msg-1", Text: "Hello", Sent: "2024-01-01T12:00:00Z"},
		ReplyMessage: Message{UUID: "msg-2", Text: "Test response", Sent: "2024-01-01T12:00:01Z"},
	})
}

// withFastRetries shrinks the retry backoff so tests don't sleep for real
// seconds, restoring the original value when the test ends.
func withFastRetries(t *testing.T) {
	t.Helper()
	orig := chatRetryBackoff
	chatRetryBackoff = time.Millisecond
	t.Cleanup(func() { chatRetryBackoff = orig })
}

func TestSendMessageRetriesOn503(t *testing.T) {
	withFastRetries(t)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		chatOKResponse(w)
	}))
	defer server.Close()

	c := NewNomiClient("test-api-key", server.URL)
	resp, err := c.SendMessage("test-uuid", "Hello")
	if err != nil {
		t.Fatalf("Expected success after retry, got error: %v", err)
	}
	if resp.ReplyMessage.Text != "Test response" {
		t.Errorf("Expected reply %q, got %q", "Test response", resp.ReplyMessage.Text)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("Expected 2 requests (1 failure + 1 retry), got %d", got)
	}
}

func TestSendMessageRetriesOnTimeout(t *testing.T) {
	withFastRetries(t)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			// Stall past the chat timeout so the first attempt is aborted.
			time.Sleep(150 * time.Millisecond)
			return
		}
		chatOKResponse(w)
	}))
	defer server.Close()

	c := NewNomiClient("test-api-key", server.URL)
	c.chatTimeout = 50 * time.Millisecond

	resp, err := c.SendMessage("test-uuid", "Hello")
	if err != nil {
		t.Fatalf("Expected success after timeout retry, got error: %v", err)
	}
	if resp.ReplyMessage.Text != "Test response" {
		t.Errorf("Expected reply %q, got %q", "Test response", resp.ReplyMessage.Text)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("Expected 2 requests (1 timeout + 1 retry), got %d", got)
	}
}

func TestSendMessageRetriesWhileNomiStillResponding(t *testing.T) {
	withFastRetries(t)

	for _, tc := range []struct {
		name      string
		statuses  []int
		wantError string
	}{
		{
			name:     "succeeds when Nomi is ready",
			statuses: []int{400, 400, 400, 400, 200},
		},
		{
			name:      "stops after four retries",
			statuses:  []int{400, 400, 400, 400, 400},
			wantError: "after 5 attempts",
		},
		{
			name:      "stops if an earlier attempt might have reached Nomi",
			statuses:  []int{503, 400},
			wantError: "NomiStillResponding",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				i := int(calls.Add(1)) - 1
				if i >= len(tc.statuses) {
					chatOKResponse(w)
					return
				}
				var req ChatRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageText != "Hello" ||
					r.Method != "POST" || r.URL.Path != "/nomis/test-uuid/chat" {
					t.Errorf("Unexpected chat request: %s %s, body %+v, decode error %v", r.Method, r.URL.Path, req, err)
				}
				switch tc.statuses[i] {
				case http.StatusBadRequest:
					http.Error(w, `{"error":{"type":"NomiStillResponding"}}`, http.StatusBadRequest)
				case http.StatusServiceUnavailable:
					http.Error(w, "service unavailable", http.StatusServiceUnavailable)
				case http.StatusOK:
					chatOKResponse(w)
				}
			}))
			defer server.Close()

			c := NewNomiClient("test-api-key", server.URL)
			resp, err := c.SendMessage("test-uuid", "Hello")
			if tc.wantError == "" {
				if err != nil || resp == nil || resp.ReplyMessage.Text != "Test response" {
					t.Errorf("Expected a reply after retries, got %+v, error %v", resp, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("Expected error containing %q, got %v", tc.wantError, err)
			}
			if got := int(calls.Load()); got != len(tc.statuses) {
				t.Errorf("Expected %d requests, got %d", len(tc.statuses), got)
			}
		})
	}
}

func TestSendMessageNoRetryOn400(t *testing.T) {
	withFastRetries(t)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, `{"error":{"type":"InvalidBody","message":"NomiStillResponding"}}`, http.StatusBadRequest)
	}))
	defer server.Close()

	c := NewNomiClient("test-api-key", server.URL)
	if _, err := c.SendMessage("test-uuid", "Hello"); err == nil {
		t.Fatal("Expected error for 400 response, got none")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("Expected 1 request for InvalidBody, got %d", got)
	}
}

func TestSendMessageFailsAfterMaxRetries(t *testing.T) {
	withFastRetries(t)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewNomiClient("test-api-key", server.URL)
	_, err := c.SendMessage("test-uuid", "Hello")
	if err == nil {
		t.Fatal("Expected error after exhausting retries, got none")
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Errorf("Expected error to mention attempt count, got: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != maxChatRetries+1 {
		t.Errorf("Expected %d requests, got %d", maxChatRetries+1, got)
	}
}
