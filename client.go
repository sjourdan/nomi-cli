package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NomiClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
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
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

func (c *NomiClient) makeRequest(method, endpoint string, body interface{}, result interface{}) error {
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

	if reqBody != nil {
		req, err = http.NewRequest(method, url, reqBody)
	} else {
		req, err = http.NewRequest(method, url, nil)
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
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &APIError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Request to %s failed", endpoint),
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
	err := c.makeRequest("GET", "/nomis", nil, &response)
	if err != nil {
		return nil, err
	}
	return response.Nomis, nil
}

func (c *NomiClient) GetNomi(id string) (*Nomi, error) {
	var nomi Nomi
	endpoint := fmt.Sprintf("/nomis/%s", id)
	err := c.makeRequest("GET", endpoint, nil, &nomi)
	if err != nil {
		return nil, err
	}
	return &nomi, nil
}

func (c *NomiClient) GetRooms() ([]Room, error) {
	var response RoomResponse
	err := c.makeRequest("GET", "/rooms", nil, &response)
	if err != nil {
		return nil, err
	}
	return response.Rooms, nil
}

func (c *NomiClient) SendMessage(nomiID, message string) (*ChatResponse, error) {
	var response ChatResponse
	endpoint := fmt.Sprintf("/nomis/%s/chat", nomiID)
	requestBody := ChatRequest{MessageText: message}
	err := c.makeRequest("POST", endpoint, requestBody, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *NomiClient) FindNomiByName(name string) (string, error) {
	nomis, err := c.GetNomis()
	if err != nil {
		return "", err
	}

	for _, nomi := range nomis {
		if nomi.Name == name {
			return nomi.UUID, nil
		}
	}

	return "", fmt.Errorf("no Nomi found with the name: %s", name)
}
