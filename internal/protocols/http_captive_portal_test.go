package protocols

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestCaptivePortalRedirectsProbePathsAndClears(t *testing.T) {
	handler, device, state := portalTestHandler()
	for _, path := range []string{"/custom", "/generate_204", "/connecttest.txt", "/unknown?next=https://outside.invalid", "//outside.invalid"} {
		t.Run(path, func(t *testing.T) {
			request := &HTTPRequest{Method: "GET", Path: path}
			baseline := readPortalResponse(
				t,
				handler.generateResponse(request, []*config.Device{device}),
			)
			if err := state.SetDeviceFault("captive_portal", 1); err != nil {
				t.Fatal(err)
			}
			response := readPortalResponse(
				t,
				handler.generateResponse(request, []*config.Device{device}),
			)
			if response.status != http.StatusFound || response.location != "/niac-portal" {
				t.Fatalf("redirect = %#v", response)
			}
			if err := state.SetDeviceFault("captive_portal", 0); err != nil {
				t.Fatal(err)
			}
			if got := readPortalResponse(
				t,
				handler.generateResponse(request, []*config.Device{device}),
			); got != baseline {
				t.Fatalf("clear = %#v, baseline = %#v", got, baseline)
			}
		})
	}
}

func TestCaptivePortalLandingAndHealthyPeer(t *testing.T) {
	handler, device, state := portalTestHandler()
	peer := &config.Device{Name: "healthy", HTTPConfig: device.HTTPConfig}
	handler.stack.deviceStates[peer] = devicestate.NewStore(
		devicestate.Identity{Hostname: peer.Name},
	)
	request := &HTTPRequest{Method: "GET", Path: "/custom"}
	baseline := readPortalResponse(t, handler.generateResponse(request, []*config.Device{peer}))
	if err := state.SetDeviceFault("captive_portal", 1); err != nil {
		t.Fatal(err)
	}
	landing := readPortalResponse(
		t,
		handler.generateResponse(
			&HTTPRequest{Method: "GET", Path: "/niac-portal"},
			[]*config.Device{device},
		),
	)
	if landing.status != http.StatusOK || landing.location != "" ||
		!strings.Contains(landing.body, "Captive portal") {
		t.Fatalf("landing = %#v", landing)
	}
	if strings.Contains(landing.body, "<form") {
		t.Fatal("portal must not collect credentials")
	}
	if got := readPortalResponse(t, handler.generateResponse(request, []*config.Device{peer})); got != baseline {
		t.Fatalf("healthy peer changed: %#v", got)
	}
}

func TestCaptivePortalHeadHasGetHeadersWithoutBody(t *testing.T) {
	handler, device, state := portalTestHandler()
	if err := state.SetDeviceFault("captive_portal", 1); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/niac-portal", "/generate_204"} {
		get := handler.generateResponse(
			&HTTPRequest{Method: "GET", Path: path},
			[]*config.Device{device},
		)
		head := handler.generateResponse(
			&HTTPRequest{Method: "HEAD", Path: path},
			[]*config.Device{device},
		)
		_, getBody, _ := bytes.Cut(get, []byte("\r\n\r\n"))
		_, headBody, found := bytes.Cut(head, []byte("\r\n\r\n"))
		if !found || len(headBody) != 0 {
			t.Fatalf("HEAD %s returned payload %q", path, headBody)
		}
		response, err := http.ReadResponse(
			bufio.NewReader(bytes.NewReader(head)),
			&http.Request{Method: "HEAD"},
		)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if got := response.Header.Get("Content-Length"); got != strconv.Itoa(len(getBody)) {
			t.Fatalf("HEAD %s Content-Length=%s, GET body length=%d", path, got, len(getBody))
		}
	}
}

func portalTestHandler() (*HTTPHandler, *config.Device, *devicestate.Store) {
	device := &config.Device{Name: "portal", HTTPConfig: &config.HTTPConfig{
		Enabled: true, ServerName: "AuthoredServer/1.0",
		Endpoints: []config.HTTPEndpoint{
			{
				Path:        "/custom",
				Method:      "GET",
				StatusCode:  201,
				ContentType: "text/plain",
				Body:        "authored content",
			},
		},
	}}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	stack := &Stack{deviceStates: map[*config.Device]*devicestate.Store{device: state}}
	return NewHTTPHandler(stack), device, state
}

type portalResponse struct {
	status                              int
	location, body, contentType, server string
}

func readPortalResponse(t *testing.T, data []byte) portalResponse {
	t.Helper()
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return portalResponse{
		response.StatusCode,
		response.Header.Get("Location"),
		string(body),
		response.Header.Get("Content-Type"),
		response.Header.Get("Server"),
	}
}
