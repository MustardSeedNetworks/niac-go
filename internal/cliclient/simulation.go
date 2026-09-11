package cliclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

const (
	csrfEndpoint         = "/api/v1/csrf-token"
	defaultSessionID     = "default"
	maxErrorResponseSize = 64 << 10
)

type csrfTokenResponse struct {
	Token string `json:"token"`
}

type mutation struct {
	method  string
	path    string
	payload []byte
	output  any
}

// SimulationStatus is the daemon's current simulation selection.
type SimulationStatus struct {
	SessionID string             `json:"sessionId"`
	Running   bool               `json:"running"`
	Sessions  []SimulationStatus `json:"sessions,omitempty"`
}

// SimulationRequest identifies a managed scenario the daemon should compile or run.
type SimulationRequest struct {
	SessionID      string                `json:"sessionId,omitempty"`
	Interface      string                `json:"interface"`
	Attachment     string                `json:"attachment,omitempty"`
	AttachmentMode fabric.AttachmentMode `json:"attachmentMode,omitempty"`
	AccessVLAN     uint16                `json:"accessVlan,omitempty"`
	ConfigPath     string                `json:"configPath,omitempty"`
	ConfigData     string                `json:"configData,omitempty"`
	TemplateName   string                `json:"templateName,omitempty"`
}

// PreflightSimulation compiles a scenario without changing daemon state.
func (c *Client) PreflightSimulation(ctx context.Context, request SimulationRequest) (*fabric.Report, error) {
	var report fabric.Report
	if err := c.mutate(ctx, http.MethodPost, "/api/v1/simulation/preflight", request, &report); err != nil {
		return nil, err
	}

	return &report, nil
}

// StartSimulation asks the daemon to start a managed scenario.
func (c *Client) StartSimulation(ctx context.Context, request SimulationRequest) (*SimulationStatus, error) {
	var status SimulationStatus
	if err := c.mutate(ctx, http.MethodPost, "/api/v1/simulation", request, &status); err != nil {
		return nil, err
	}
	targetSession := request.SessionID
	if targetSession == "" {
		targetSession = defaultSessionID
	}
	for index := range status.Sessions {
		if status.Sessions[index].SessionID == targetSession {
			return &status.Sessions[index], nil
		}
	}
	return &status, nil
}

// SelectSimulation makes sessionID the daemon's selected simulation.
func (c *Client) SelectSimulation(ctx context.Context, sessionID string) (*SimulationStatus, error) {
	request := struct {
		SessionID string `json:"sessionId"`
	}{SessionID: sessionID}
	var status SimulationStatus
	if err := c.mutate(ctx, http.MethodPut, "/api/v1/simulation", request, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// StopSimulation asks the daemon to stop one scenario by session ID.
func (c *Client) StopSimulation(ctx context.Context, sessionID string) error {
	path := "/api/v1/sessions/" + url.PathEscape(sessionID)
	return c.mutate(ctx, http.MethodDelete, path, struct{}{}, nil)
}

func (c *Client) mutate(ctx context.Context, method, path string, input, output any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode %s request: %w", path, err)
	}
	for range 2 {
		csrfToken, tokenErr := c.csrfToken(ctx)
		if tokenErr != nil {
			return tokenErr
		}
		request := mutation{method: method, path: path, payload: payload, output: output}
		retry, requestErr := c.sendMutation(ctx, request, csrfToken)
		if !retry || requestErr == nil {
			return requestErr
		}
	}

	return fmt.Errorf("%w: %s rejected a refreshed CSRF token", ErrRequestFailed, path)
}

func (c *Client) csrfToken(ctx context.Context) (string, error) {
	var response csrfTokenResponse
	if err := c.get(ctx, csrfEndpoint, &response); err != nil {
		return "", fmt.Errorf("get daemon CSRF token: %w", err)
	}
	if response.Token == "" {
		return "", fmt.Errorf("%w: daemon returned an empty CSRF token", ErrRequestFailed)
	}

	return response.Token, nil
}

func (c *Client) sendMutation(ctx context.Context, mutation mutation, csrfToken string) (bool, error) {
	req, err := http.NewRequestWithContext(
		ctx, mutation.method, c.baseURL+mutation.path, bytes.NewReader(mutation.payload),
	)
	if err != nil {
		return false, fmt.Errorf("build request for %s: %w", mutation.path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", csrfToken)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, c.transportError(mutation.path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return false, decodeMutationResponse(resp.Body, mutation.path, mutation.output)
	}
	return classifyMutationError(resp, mutation.path)
}

func decodeMutationResponse(body io.Reader, path string, output any) error {
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(body).Decode(output); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func classifyMutationError(resp *http.Response, path string) (bool, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
	if err != nil {
		return false, fmt.Errorf("read %s response: %w", path, err)
	}
	csrfRejected := strings.Contains(string(body), "csrf_token_invalid") ||
		strings.Contains(string(body), "csrf_token_expired")
	if resp.StatusCode == http.StatusForbidden && csrfRejected {
		return true, fmt.Errorf("%w: stale CSRF token", ErrRequestFailed)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, fmt.Errorf("%w: set NIAC_API_TOKEN", ErrUnauthorized)
	}
	if resp.StatusCode == http.StatusForbidden {
		return false, ErrForbidden
	}
	return false, fmt.Errorf("%w: %s returned %s: %s", ErrRequestFailed, path, resp.Status, body)
}
