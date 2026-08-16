// Package oui provides deterministic vendor MAC allocation from IEEE data.
package oui

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

//go:embed data/oui.txt
var embeddedData string

var (
	// ErrUnknownVendor means the requested vendor name matches no IEEE MA-L
	// (organizationally unique identifier) assignment in the embedded registry.
	ErrUnknownVendor = errors.New("vendor has no matching IEEE MA-L assignment")
	// ErrOrdinalRange means a requested MAC suffix does not fit the 24-bit
	// range available after a vendor's 3-byte OUI prefix.
	ErrOrdinalRange = errors.New("MAC suffix must fit in 24 bits")
)

type prefix [3]byte

type organization struct {
	name       string
	normalized string
}

type prefixMatch struct {
	prefix prefix
	found  bool
}

const (
	prefixBytes     = 3
	maxSuffix       = 0xffffff
	middleByteShift = 8
	highByteShift   = 2 * middleByteShift
)

// Registry indexes IEEE MA-L assignments by their 24-bit prefix.
type Registry struct {
	organizations map[prefix]organization
	matchMu       sync.RWMutex
	matches       map[string]prefixMatch
}

// LoadEmbedded parses the IEEE snapshot shipped with NIAC.
func LoadEmbedded() (*Registry, error) {
	return Parse(strings.NewReader(embeddedData))
}

// Parse reads the public IEEE oui.txt format.
func Parse(reader io.Reader) (*Registry, error) {
	registry := &Registry{
		organizations: make(map[prefix]organization),
		matches:       make(map[string]prefixMatch),
	}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		parsedPrefix, parsedOrganization, ok := parseLine(scanner.Text())
		if ok {
			registry.organizations[parsedPrefix] = organization{
				name: parsedOrganization, normalized: strings.ToLower(parsedOrganization),
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read IEEE OUI data: %w", err)
	}
	return registry, nil
}

// Lookup returns the IEEE organization for a MAC address.
func (r *Registry) Lookup(mac net.HardwareAddr) (string, bool) {
	if len(mac) < prefixBytes {
		return "", false
	}
	entry, ok := r.organizations[prefix{mac[0], mac[1], mac[2]}]
	return entry.name, ok
}

// Allocate returns a deterministic isolated-lab MAC for a vendor and suffix.
func (r *Registry) Allocate(vendor string, ordinal uint32) (net.HardwareAddr, error) {
	if ordinal > maxSuffix {
		return nil, ErrOrdinalRange
	}
	selected, found := r.matchingPrefix(vendorSearch(vendor))
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrUnknownVendor, vendor)
	}
	return net.HardwareAddr{
		selected[0], selected[1], selected[2],
		safeconv.ByteFromUint32(ordinal >> highByteShift),
		safeconv.ByteFromUint32(ordinal >> middleByteShift),
		safeconv.ByteFromUint32(ordinal),
	}, nil
}

func (r *Registry) matchingPrefix(search string) (prefix, bool) {
	r.matchMu.RLock()
	match, cached := r.matches[search]
	r.matchMu.RUnlock()
	if cached {
		return match.prefix, match.found
	}

	var selected prefix
	found := false
	for candidate, organization := range r.organizations {
		if strings.Contains(organization.normalized, search) &&
			(!found || comparePrefix(candidate, selected) < 0) {
			selected = candidate
			found = true
		}
	}

	r.matchMu.Lock()
	if r.matches == nil {
		r.matches = make(map[string]prefixMatch)
	}
	if existing, ok := r.matches[search]; ok {
		match = existing
	} else {
		match = prefixMatch{prefix: selected, found: found}
		r.matches[search] = match
	}
	r.matchMu.Unlock()
	return match.prefix, match.found
}

func parseLine(line string) (prefix, string, bool) {
	prefixText, organization, found := strings.Cut(line, "(hex)")
	if !found {
		return prefix{}, "", false
	}
	raw := strings.ReplaceAll(strings.TrimSpace(prefixText), "-", "")
	mac, err := hex.DecodeString(raw)
	if err != nil || len(mac) != 3 {
		return prefix{}, "", false
	}
	organization = strings.TrimSpace(organization)
	if organization == "" {
		return prefix{}, "", false
	}
	return prefix{mac[0], mac[1], mac[2]}, organization, true
}

func vendorSearch(vendor string) string {
	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "aruba":
		return "hewlett packard enterprise"
	case "meraki":
		return "cisco meraki"
	case "mikrotik":
		return "routerboard.com"
	case "palo alto":
		return "palo alto networks"
	case "raspberry pi":
		return "raspberry pi"
	default:
		return strings.ToLower(strings.TrimSpace(vendor))
	}
}

func comparePrefix(left, right prefix) int {
	return bytes.Compare(left[:], right[:])
}
