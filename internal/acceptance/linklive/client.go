// Package linklive provides development-only access to Link-Live analysis data.
package linklive

import (
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
		`"displayedDeviceType":1,"analysisId":1,"interfaces":1}`
)

var analysisIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var errTokenRefreshRequired = errors.New("Link-Live access token requires refresh")

// ErrMFARequired means Link-Live requires an OTP for this login.
var ErrMFARequired = errors.New("Link-Live login requires MFA")

// Config contains credentials and endpoint overrides for acceptance testing.
type Config struct {
	IdentityURL    string
	APIURL         string
	Username       string
	Password       string
	MFACode        string
	OrganizationID string
	AccessToken    string
	HTTPClient     *http.Client
	AllowInsecure  bool
}

// Client reads Link-Live analysis data without mutating it.
type Client struct {
	config Config
	mu     sync.Mutex
	token  string
}

// New validates configuration and constructs a Link-Live client.
func New(config Config) (*Client, error) {
	if config.AccessToken == "" && (strings.TrimSpace(config.Username) == "" || config.Password == "") {
		return nil, errors.New("Link-Live username and password are required")
	}
	if err := validateBaseURL(config.IdentityURL, config.AllowInsecure); err != nil {
		return nil, fmt.Errorf("identity URL: %w", err)
	}
	if err := validateBaseURL(config.APIURL, config.AllowInsecure); err != nil {
		return nil, fmt.Errorf("API URL: %w", err)
	}
	config.HTTPClient = redirectSafeClient(config.HTTPClient)
	return &Client{config: config, token: config.AccessToken}, nil
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
	data, err := c.authorizedGet(ctx, path, token)
	if !errors.Is(err, errTokenRefreshRequired) {
		return data, err
	}
	token, err = c.refreshAccessToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return c.authorizedGet(ctx, path, token)
}

func (c *Client) authorizedGet(
	ctx context.Context,
	path string,
	token string,
) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.APIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create Link-Live request: %w", err)
	}
	req.Header.Set("Authorization", "Access "+token)
	return c.do(req)
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
	if resp.StatusCode == http.StatusUpgradeRequired {
		return nil, errTokenRefreshRequired
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
