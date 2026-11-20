package cliclient

import (
	"context"
	"encoding/base64"
	"net/http"
)

// LibraryInstallRequest mirrors api.LibraryInstallRequest: a gzip-tar content
// bundle, base64-encoded so it travels as an ordinary JSON body.
type LibraryInstallRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"`
	Force    bool   `json:"force,omitempty"`
	DryRun   bool   `json:"dryRun,omitempty"`
}

// LibraryInstallResult is what the daemon installed, mirroring
// api.LibraryInstallResponse.
type LibraryInstallResult struct {
	Success     bool           `json:"success"`
	Files       int            `json:"files"`
	Directories int            `json:"directories"`
	Bytes       int64          `json:"bytes"`
	PerKind     map[string]int `json:"perKind"`
	Preserved   int            `json:"preserved"`
	Message     string         `json:"message"`
}

// InstallPack hands a content bundle to a running daemon, which owns its
// library and is the only thing that writes to it while it runs.
//
// The endpoint is admin-scoped, so a daemon bound anywhere but loopback
// answers ErrForbidden to a read-write token.
func (c *Client) InstallPack(
	ctx context.Context,
	filename string,
	bundle []byte,
	force, dryRun bool,
) (*LibraryInstallResult, error) {
	request := LibraryInstallRequest{
		Filename: filename,
		Data:     base64.StdEncoding.EncodeToString(bundle),
		Force:    force,
		DryRun:   dryRun,
	}

	var result LibraryInstallResult
	if err := c.mutate(ctx, http.MethodPost, "/api/v1/library/install", request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

type libraryNetworkSave struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// NetworkExists reports whether the daemon's library already holds a network
// by this name. A save replaces silently, so a caller that would clobber
// someone's scenario asks first.
func (c *Client) NetworkExists(ctx context.Context, name string) (bool, error) {
	var entries []struct {
		Name string `json:"name"`
	}
	if err := c.get(ctx, "/api/v1/library/networks", &entries); err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// SaveNetwork stores scenario YAML in the daemon's library under name. The
// daemon validates the document before keeping it, so a scenario it would
// refuse to start never lands in the library.
func (c *Client) SaveNetwork(ctx context.Context, name, content string) error {
	return c.mutate(ctx, http.MethodPost, "/api/v1/library/networks",
		libraryNetworkSave{Name: name, Content: content}, nil)
}
