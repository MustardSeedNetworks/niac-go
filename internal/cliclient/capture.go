package cliclient

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// CaptureExportOptions narrows what ExportCapture asks for.
type CaptureExportOptions struct {
	// Filter is a libpcap BPF expression the daemon compiles and applies to
	// the retained frames. Empty means every frame.
	Filter string
	// Last keeps only the newest N frames. Zero means every frame.
	Last int
}

// ExportCapture writes the session's retained frames to out as pcapng and
// reports how many bytes were written.
//
// The daemon keeps a bounded ring per session, so this is the recent window
// rather than the whole run — but unlike the packet stream it is there before
// the caller connects, and unlike the stream's events the frames are whole.
func (c *Client) ExportCapture(
	ctx context.Context, sessionID string, options CaptureExportOptions, out io.Writer,
) (int64, error) {
	query := url.Values{}
	if options.Filter != "" {
		query.Set("filter", options.Filter)
	}
	if options.Last > 0 {
		query.Set("last", strconv.Itoa(options.Last))
	}
	path := "/api/v1/sessions/" + url.PathEscape(sessionID) + "/capture/export"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	body, err := c.open(ctx, path)
	if err != nil {
		return 0, err
	}
	defer body.Close()

	written, err := io.Copy(out, body)
	if err != nil {
		return written, fmt.Errorf("read capture export: %w", err)
	}

	return written, nil
}
