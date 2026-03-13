package sonar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("Sonar API request failed with status %d", e.StatusCode)
	}
	return e.Message
}

type apiErrors struct {
	Errors []struct {
		Message string `json:"msg"`
	} `json:"errors"`
}

func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse host URL: %w", err)
	}

	return &Client{
		baseURL: parsedURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		token: token,
	}, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, out any) error {
	relativeURL := &url.URL{Path: path}
	if len(query) > 0 {
		relativeURL.RawQuery = query.Encode()
	}
	requestURL := c.baseURL.ResolveReference(relativeURL)

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeHTTPError(resp.StatusCode, body)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func decodeHTTPError(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))

	var apiErr apiErrors
	if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 && apiErr.Errors[0].Message != "" {
		message = apiErr.Errors[0].Message
	}

	if message == "" {
		message = fmt.Sprintf("Sonar API request failed with status %d", statusCode)
	}

	return &HTTPError{StatusCode: statusCode, Message: message}
}
