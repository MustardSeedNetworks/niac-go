package cliclient

import "context"

// BuildVersion is what /__version reports. A release is only proven by the
// three fields the build contract injects: an empty UIBuildHash means the
// binary was built outside the make pipeline and does not embed the UI.
type BuildVersion struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	CommitFull  string `json:"commitFull"`
	BuildTime   string `json:"buildTime"`
	UIBuildHash string `json:"uiBuildHash"`
	Platform    string `json:"platform"`
}

// Version reads the daemon's build information. The route needs no token, so
// this also answers on a daemon whose token this client does not hold.
func (c *Client) Version(ctx context.Context) (*BuildVersion, error) {
	var version BuildVersion
	if err := c.get(ctx, "/__version", &version); err != nil {
		return nil, err
	}

	return &version, nil
}
