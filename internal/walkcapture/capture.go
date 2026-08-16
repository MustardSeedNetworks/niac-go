// Package walkcapture performs bounded SNMP walks without invoking a command
// that would expose credentials in the process list.
package walkcapture

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	niacsnmp "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

const (
	defaultTimeout = 30 * time.Second
	maximumTimeout = 60 * time.Second
	maximumEntries = 100_000
	maximumBytes   = 16 << 20
	defaultPort    = uint16(161)
	requestTimeout = 5 * time.Second
	requestRetries = 1
	bulkMaxRepeat  = uint32(25)
	maxCommunity   = 255
)

var (
	// ErrInvalidRequest means the Request failed validation (target, timeout,
	// credentials, or privacy protocol) before capture could begin.
	ErrInvalidRequest = errors.New("invalid walk capture request")
	// ErrEntryLimit means the capture stopped after reaching maximumEntries.
	ErrEntryLimit = errors.New("walk capture exceeded 100000 entries")
	// ErrSizeLimit means the capture stopped after reaching maximumBytes.
	ErrSizeLimit = errors.New("walk capture exceeded 16 MiB")
)

// Request contains request-only SNMP connection material. Callers must not
// persist, return, or log this value.
type Request struct {
	Target       string `json:"target"`
	Port         uint16 `json:"port"`
	Version      string `json:"version"`
	Community    string `json:"community,omitempty"`
	Username     string `json:"username,omitempty"`
	AuthProtocol string `json:"authProtocol,omitempty"`
	AuthPassword string `json:"authPassword,omitempty"`
	PrivProtocol string `json:"privProtocol,omitempty"`
	PrivPassword string `json:"privPassword,omitempty"`
	TimeoutSecs  int    `json:"timeoutSeconds,omitempty"`
}

// Capture walks the internet subtree and returns numeric net-snmp content.
// The request context and bounded timeout both cancel in-flight network I/O.
func Capture(ctx context.Context, request Request) ([]byte, error) {
	client, timeout, err := clientFor(ctx, request)
	if err != nil {
		return nil, err
	}
	captureContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client.Context = captureContext
	if err = client.Connect(); err != nil {
		return nil, fmt.Errorf("connect to SNMP target: %w", err)
	}
	defer func() { _ = client.Conn.Close() }()

	var content bytes.Buffer
	entryCount := 0
	for _, root := range []string{".1.3.6.1", ".1.0.8802.1.1.2"} {
		err = client.BulkWalk(root, func(pdu gosnmp.SnmpPDU) error {
			if entryCount >= maximumEntries {
				return ErrEntryLimit
			}
			line := niacsnmp.FormatWalkEntries(
				[]niacsnmp.WalkEntry{{OID: pdu.Name, Type: pdu.Type, Value: pdu.Value}},
			)
			if content.Len()+len(line) > maximumBytes {
				return ErrSizeLimit
			}
			entryCount++
			_, _ = content.Write(line)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("capture SNMP walk: %w", err)
		}
	}
	if entryCount == 0 {
		return nil, errors.New("capture SNMP walk: no variables returned")
	}
	return content.Bytes(), nil
}

func clientFor(ctx context.Context, request Request) (*gosnmp.GoSNMP, time.Duration, error) {
	if _, err := netip.ParseAddr(request.Target); err != nil {
		return nil, 0, fmt.Errorf("%w: target must be an IP address", ErrInvalidRequest)
	}
	port := request.Port
	if port == 0 {
		port = defaultPort
	}
	timeout := defaultTimeout
	if request.TimeoutSecs != 0 {
		timeout = time.Duration(request.TimeoutSecs) * time.Second
	}
	if timeout < time.Second || timeout > maximumTimeout {
		return nil, 0, fmt.Errorf("%w: timeout must be between 1 and 60 seconds", ErrInvalidRequest)
	}

	client := &gosnmp.GoSNMP{
		Target: request.Target, Port: port, Transport: "udp", Context: ctx,
		Timeout: min(requestTimeout, timeout), Retries: requestRetries, MaxRepetitions: bulkMaxRepeat,
	}
	switch strings.ToLower(strings.TrimSpace(request.Version)) {
	case "2c":
		if request.Community == "" || len(request.Community) > maxCommunity {
			return nil, 0, fmt.Errorf("%w: SNMPv2c community is required", ErrInvalidRequest)
		}
		client.Version = gosnmp.Version2c
		client.Community = request.Community
	case "3":
		if err := configureV3(client, request); err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, fmt.Errorf("%w: version must be 2c or 3", ErrInvalidRequest)
	}
	return client, timeout, nil
}
