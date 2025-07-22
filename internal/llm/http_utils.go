package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient is a configurable HTTP client for LLM API calls
type HTTPClient struct {
	client  *http.Client
	headers map[string]string
}

// HTTPConfig holds configuration for HTTP requests
type HTTPConfig struct {
	Timeout time.Duration
	Headers map[string]string
}

// NewHTTPClient creates a new HTTPClient with the given configuration
func NewHTTPClient(config HTTPConfig) *HTTPClient {
	if config.Timeout == 0 {
		config.Timeout = DefaultHTTPTimeout
	}
	
	return &HTTPClient{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		headers: config.Headers,
	}
}

// PostJSON makes a POST request with JSON payload
func (h *HTTPClient) PostJSON(url string, payload interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	
	// Set custom headers
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}

	return resp, nil
}

// ReadResponseBody reads and returns the response body as string
func ReadResponseBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(ErrFailedToReadResponse, err)
	}
	
	return string(body), nil
}

// CheckStatusCode validates HTTP status code and returns error if not successful
func CheckStatusCode(resp *http.Response, body string) error {
	if resp.StatusCode >= 400 {
		return fmt.Errorf(ErrAPIRequestFailed, resp.StatusCode, body)
	}
	return nil
}

// UnmarshalJSONResponse reads response body and unmarshals JSON into target
func UnmarshalJSONResponse(resp *http.Response, target interface{}) error {
	body, err := ReadResponseBody(resp)
	if err != nil {
		return err
	}
	
	if err := CheckStatusCode(resp, body); err != nil {
		return err
	}
	
	if err := json.Unmarshal([]byte(body), target); err != nil {
		return fmt.Errorf(ErrFailedToDecodeResponse, err)
	}
	
	return nil
}