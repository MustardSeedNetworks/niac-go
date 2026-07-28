// Package linklive provides development-only access to Link-Live analysis data.
package linklive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

const (
	linkLiveAppClaim   = "3pGv3iSAICObaFX7EjdojO"
	maxResponseBytes   = 32 << 20
	mfaRequiredStatus  = 498
	topologyProjection = `{"bestNameFormatted":1,"longMfrMac":1,"defaultAddr.ipV4Address":1,` +
		`"comment":1,"_id":1,"addresses":1,"connectedHosts":1,"hostIds":1,` +
		`"hostId":1,"change":1,"monitoring":1,"worstProblem":1,` +
		`"displayedDeviceType":1,"analysisId":1}`
)

var analysisIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ErrMFARequired means Link-Live requires an OTP for this login.
var ErrMFARequired = errors.New("Link-Live login requires MFA")

// Config contains credentials and endpoint overrides for acceptance testing.
type Config struct {
	IdentityURL   string
	APIURL        string
	Username      string
	Password      string
	MFACode       string
	HTTPClient    *http.Client
	AllowInsecure bool
}

// Client reads Link-Live analysis data without mutating it.
type Client struct {
	config Config
	mu     sync.Mutex
	token  string
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

// New validates configuration and constructs a Link-Live client.
func New(config Config) (*Client, error) {
	if strings.TrimSpace(config.Username) == "" || config.Password == "" {
		return nil, errors.New("Link-Live username and password are required")
	}
	if err := validateBaseURL(config.IdentityURL, config.AllowInsecure); err != nil {
		return nil, fmt.Errorf("identity URL: %w", err)
	}
	if err := validateBaseURL(config.APIURL, config.AllowInsecure); err != nil {
		return nil, fmt.Errorf("API URL: %w", err)
	}
	config.HTTPClient = redirectSafeClient(config.HTTPClient)
	return &Client{config: config}, nil
}

// Analyses returns the unmodified analysis-list response.
func (c *Client) Analyses(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/v1/admin/analysis")
}

// Topology returns the projected hosts and inferred connections for an analysis.
func (c *Client) Topology(ctx context.Context, analysisID string) (json.RawMessage, error) {
	if !analysisIDPattern.MatchString(analysisID) {
		return nil, errors.New("analysis ID contains invalid characters")
	}
	query, err := json.Marshal(map[string]string{"analysisId": analysisID})
	if err != nil {
		return nil, errors.New("encode topology query")
	}
	values := url.Values{"query": {string(query)}, "project": {topologyProjection}}
	return c.get(ctx, "/v1/admin/hosts?"+values.Encode())
}

func (c *Client) get(ctx context.Context, path string) (json.RawMessage, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.APIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create Link-Live request: %w", err)
	}
	req.Header.Set("Authorization", "Access "+token)
	return c.do(req)
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" {
		return c.token, nil
	}
	token, err := c.login(ctx)
	if err == nil {
		c.token = token
	}
	return token, err
}

func (c *Client) login(ctx context.Context) (string, error) {
	body := []byte(`{"claims":{"app":"` + linkLiveAppClaim + `"}}`)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.config.IdentityURL+"/v2/auth/login",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create Link-Live login request: %w", err)
	}
	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("Content-Type", "application/json")
	if c.config.MFACode != "" {
		req.Header.Set("X-Bedrock-Otpcode", c.config.MFACode)
	}
	data, err := c.do(req)
	if err != nil {
		return "", err
	}
	return decodeToken(data)
}

func (c *Client) do(req *http.Request) (json.RawMessage, error) {
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Link-Live request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == mfaRequiredStatus {
		return nil, ErrMFARequired
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Link-Live returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Link-Live response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("Link-Live response exceeds size limit")
	}
	return data, nil
}

func decodeToken(data []byte) (string, error) {
	var response loginResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", errors.New("Link-Live login returned invalid JSON")
	}
	if response.AccessToken == "" {
		return "", errors.New("Link-Live login response omitted access token")
	}
	return response.AccessToken, nil
}

func validateBaseURL(rawURL string, allowInsecure bool) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("must be an absolute HTTP or HTTPS URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("must use HTTP or HTTPS")
	}
	if parsed.Scheme != "https" && !allowInsecure {
		return errors.New("must use HTTPS")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("must not include a query or fragment")
	}
	return nil
}

func redirectSafeClient(source *http.Client) *http.Client {
	if source == nil {
		source = http.DefaultClient
	}
	client := *source
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}
