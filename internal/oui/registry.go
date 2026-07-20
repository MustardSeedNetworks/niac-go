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
	"slices"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

//go:embed data/oui.txt
var embeddedData string

var (
	ErrUnknownVendor = errors.New("vendor has no matching IEEE MA-L assignment")
	ErrOrdinalRange  = errors.New("MAC suffix must fit in 24 bits")
)

type prefix [3]byte

const (
	prefixBytes     = 3
	maxSuffix       = 0xffffff
	middleByteShift = 8
	highByteShift   = 2 * middleByteShift
)

// Registry indexes IEEE MA-L assignments by their 24-bit prefix.
type Registry struct {
	organizations map[prefix]string
}

// LoadEmbedded parses the IEEE snapshot shipped with NIAC.
func LoadEmbedded() (*Registry, error) {
	return Parse(strings.NewReader(embeddedData))
}

// Parse reads the public IEEE oui.txt format.
func Parse(reader io.Reader) (*Registry, error) {
	registry := &Registry{organizations: make(map[prefix]string)}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		parsedPrefix, organization, ok := parseLine(scanner.Text())
		if ok {
			registry.organizations[parsedPrefix] = organization
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
	organization, ok := r.organizations[prefix{mac[0], mac[1], mac[2]}]
	return organization, ok
}

// Allocate returns a deterministic isolated-lab MAC for a vendor and suffix.
func (r *Registry) Allocate(vendor string, ordinal uint32) (net.HardwareAddr, error) {
	if ordinal > maxSuffix {
		return nil, ErrOrdinalRange
	}
	prefixes := r.matchingPrefixes(vendorSearch(vendor))
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrUnknownVendor, vendor)
	}
	selected := prefixes[0]
	return net.HardwareAddr{
		selected[0], selected[1], selected[2],
		safeconv.ByteFromUint32(ordinal >> highByteShift),
		safeconv.ByteFromUint32(ordinal >> middleByteShift),
		safeconv.ByteFromUint32(ordinal),
	}, nil
}

func (r *Registry) matchingPrefixes(search string) []prefix {
	matches := make([]prefix, 0)
	for candidate, organization := range r.organizations {
		if strings.Contains(strings.ToLower(organization), search) {
			matches = append(matches, candidate)
		}
	}
	slices.SortFunc(matches, comparePrefix)
	return matches
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
	if strings.EqualFold(strings.TrimSpace(vendor), "aruba") {
		return "hewlett packard enterprise"
	}
	return strings.ToLower(strings.TrimSpace(vendor))
}

func comparePrefix(left, right prefix) int {
	return bytes.Compare(left[:], right[:])
}
