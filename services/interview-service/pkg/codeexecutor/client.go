package codeexecutor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CodeExecutionRequest struct {
	Language  string        `json:"language"`
	Code      string        `json:"code"`
	Input     string        `json:"input,omitempty"`
	TestCases []TestCase    `json:"test_cases,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	Args      []string      `json:"args,omitempty"`
}

type TestCase struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

type TestResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

type CodeExecutionResult struct {
	Status      string        `json:"status"`
	Output      string        `json:"output"`
	Error       string        `json:"error,omitempty"`
	Runtime     time.Duration `json:"runtime"`
	Memory      int64         `json:"memory"`
	ExitCode    int           `json:"exit_code"`
	TestResults []TestResult  `json:"test_results,omitempty"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		trimmed = "http://code-executor-service:8083"
	}

	return &Client{
		baseURL: trimmed,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Execute(ctx context.Context, reqPayload *CodeExecutionRequest) (*CodeExecutionResult, error) {
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/execute", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code executor returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	result := &CodeExecutionResult{}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
