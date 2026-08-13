package cliclient

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// sseDataPrefix is the field the daemon's event stream carries its payload in;
// the id and event fields around it are framing this client does not need.
const sseDataPrefix = "data: "

// stream follows a server-sent event stream, handing each payload to onEvent
// until the context ends or onEvent asks to stop by returning false.
//
// The daemon frames events as "id: N\ndata: {json}\n\n", plus one
// "event: connected" on open, which carries no record and is skipped.
func (c *Client) stream(
	ctx context.Context,
	path string,
	onEvent func(payload []byte) bool,
) error {
	body, err := c.open(ctx, path)
	if err != nil {
		return err
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, streamBufferInitial), streamBufferMax)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, sseDataPrefix)
		if isConnectHandshake(payload) {
			continue
		}
		if !onEvent([]byte(payload)) {
			return nil
		}
	}
	if err = scanner.Err(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}

const (
	streamBufferInitial = 64 * 1024
	// A packet event carries a hex dump, so the ceiling is generous.
	streamBufferMax = 4 * 1024 * 1024
)

// isConnectHandshake reports the opening frame, which announces the stream
// rather than carrying a record.
func isConnectHandshake(payload string) bool {
	var frame struct {
		Stream string `json:"stream"`
	}
	if json.Unmarshal([]byte(payload), &frame) != nil {
		return false
	}

	return frame.Stream != "" && !strings.Contains(payload, "\"message\"") &&
		!strings.Contains(payload, "\"data\"")
}

// LogEvent is one log record from the daemon's stream.
type LogEvent struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Device    string `json:"device"`
	Source    string `json:"source"`
	Protocol  string `json:"protocol"`
}

// StreamLogs follows the daemon's log stream. onLog returning false ends it.
func (c *Client) StreamLogs(ctx context.Context, onLog func(LogEvent) bool) error {
	return c.stream(ctx, "/api/v1/stream/logs", func(payload []byte) bool {
		var event LogEvent
		if json.Unmarshal(payload, &event) != nil {
			return true
		}

		return onLog(event)
	})
}

// PacketEvent is one frame from the daemon's packet stream. The daemon sends
// the bytes hex-encoded, and truncates a frame past its wire limit rather than
// dropping it, which the reader has to be able to say out loud.
type PacketEvent struct {
	Timestamp string `json:"timestamp"`
	Direction string `json:"direction"`
	Size      int    `json:"size"`
	RawData   string `json:"raw_data"`
	Truncated bool   `json:"truncated"`
	Serial    uint64 `json:"serial"`
	Protocol  string `json:"protocol"`
	Device    string `json:"device"`
}

// Bytes decodes the frame the daemon captured.
func (p PacketEvent) Bytes() ([]byte, error) {
	raw, err := hex.DecodeString(p.RawData)
	if err != nil {
		return nil, fmt.Errorf("decode packet %d: %w", p.Serial, err)
	}

	return raw, nil
}

// StreamPackets follows the daemon's packet stream. onPacket returning false
// ends it.
func (c *Client) StreamPackets(ctx context.Context, onPacket func(PacketEvent) bool) error {
	return c.stream(ctx, "/api/v1/stream/packets", func(payload []byte) bool {
		var event PacketEvent
		if json.Unmarshal(payload, &event) != nil {
			return true
		}

		return onPacket(event)
	})
}
