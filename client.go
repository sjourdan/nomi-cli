package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Default per-request timeouts. Chat is generous because the /chat endpoint
// triggers LLM generation on Nomi's side, which routinely exceeds 30s.
const (
	defaultRequestTimeout = 30 * time.Second
	defaultChatTimeout    = 120 * time.Second
)

// Retry policy for chat requests, which can fail transiently (timeouts,
// 5xx responses). maxChatRetries is the number of retries after the first
// attempt; backoff doubles each retry starting from chatRetryBackoff.
const maxChatRetries = 2

// chatRetryBackoff is the delay before the first retry; it doubles for each
// subsequent retry. Declared as a var so tests can shrink it.
var chatRetryBackoff = 2 * time.Second

type NomiClient struct {
	httpClient     *http.Client
	apiKey         string
	baseURL        string
	requestTimeout time.Duration
	chatTimeout    time.Duration
}

// envDuration reads a timeout (in seconds) from an environment variable,
// falling back to def if unset or invalid.
func envDuration(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if secs, err := time.ParseDuration(v + "s"); err == nil {
			return secs
		}
	}
	return def
}

type APIError struct {
	Status     string
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d %s): %s", e.StatusCode, e.Status, e.Message)
}

func NewNomiClient(apiKey, baseURL string) *NomiClient {
	return &NomiClient{
		// No Client.Timeout: each request is bounded by its own context
		// so chat can wait longer than other calls.
		httpClient:     &http.Client{},
		apiKey:         apiKey,
		baseURL:        baseURL,
		requestTimeout: envDuration("NOMI_API_TIMEOUT", defaultRequestTimeout),
		chatTimeout:    envDuration("NOMI_CHAT_TIMEOUT", defaultChatTimeout),
	}
}

func (c *NomiClient) makeRequest(method, endpoint string, body interface{}, result interface{}, timeout time.Duration) error {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	var req *http.Request
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, reqBody)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("request to %s timed out after %s (try raising NOMI_CHAT_TIMEOUT): %w", endpoint, timeout, ctx.Err())
		}
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorMessage string
		if bodyBytes, err := io.ReadAll(resp.Body); err == nil && len(bodyBytes) > 0 {
			errorMessage = fmt.Sprintf("Request to %s failed: %s", endpoint, string(bodyBytes))
		} else {
			errorMessage = fmt.Sprintf("Request to %s failed", endpoint)
		}
		
		return &APIError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Message:    errorMessage,
		}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}

	return nil
}

func (c *NomiClient) GetNomis() ([]Nomi, error) {
	var response NomiResponse
	err := c.makeRequest("GET", "/nomis", nil, &response, c.requestTimeout)
	if err != nil {
		return nil, err
	}
	return response.Nomis, nil
}

func (c *NomiClient) GetNomi(id string) (*Nomi, error) {
	var nomi Nomi
	endpoint := fmt.Sprintf("/nomis/%s", id)
	err := c.makeRequest("GET", endpoint, nil, &nomi, c.requestTimeout)
	if err != nil {
		return nil, err
	}
	return &nomi, nil
}

func (c *NomiClient) GetRooms() ([]Room, error) {
	var response RoomResponse
	err := c.makeRequest("GET", "/rooms", nil, &response, c.requestTimeout)
	if err != nil {
		return nil, err
	}
	return response.Rooms, nil
}

// isRetryable reports whether err is a transient failure worth retrying:
// a request timeout or a 5xx server response.
func isRetryable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500
	}
	return false
}

func (c *NomiClient) SendMessage(nomiID, message string) (*ChatResponse, error) {
	endpoint := fmt.Sprintf("/nomis/%s/chat", nomiID)
	requestBody := ChatRequest{MessageText: message}

	var lastErr error
	for attempt := 0; attempt <= maxChatRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(chatRetryBackoff << (attempt - 1))
		}

		var response ChatResponse
		err := c.makeRequest("POST", endpoint, requestBody, &response, c.chatTimeout)
		if err == nil {
			return &response, nil
		}

		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("chat request failed after %d attempts: %w", maxChatRetries+1, lastErr)
}

func (c *NomiClient) FindNomiByName(name string) (string, error) {
	nomis, err := c.GetNomis()
	if err != nil {
		return "", err
	}

	for _, nomi := range nomis {
		if strings.EqualFold(nomi.Name, name) {
			return nomi.UUID, nil
		}
	}

	return "", fmt.Errorf("no Nomi found with the name: %s", name)
}
