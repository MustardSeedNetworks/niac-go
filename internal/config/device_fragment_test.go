package config_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const fragmentConfig = `# operator's note, must survive an edit
devices:
  - name: api-router
    type: router
    ips: ["10.10.0.1"]

  - name: core-switch
    type: switch
networks:
  - name: lan
`

func splice(t *testing.T, source, device, replacement string) string {
	t.Helper()

	fragment, ok := config.FindDeviceFragment(source, device)
	if !ok {
		t.Fatalf("no fragment for %q", device)
	}

	return config.SpliceDeviceFragment(source, fragment, replacement)
}

func TestSpliceDeviceFragmentPreservesEverythingElse(t *testing.T) {
	next := splice(t, fragmentConfig, "api-router", "name: api-router\ntype: firewall\n")

	for _, want := range []string{
		"# operator's note, must survive an edit",
		"  - name: api-router\n    type: firewall",
		"  - name: core-switch",
		"networks:",
	} {
		if !strings.Contains(next, want) {
			t.Errorf("spliced config lost %q:\n%s", want, next)
		}
	}

	if strings.Contains(next, "10.10.0.1") {
		t.Errorf("edited device kept a replaced field:\n%s", next)
	}
}

// The comment an operator writes inside a device block is the thing a
// re-serialising save silently deletes, which is why the splice exists.
func TestSpliceDeviceFragmentKeepsACommentInsideTheEditedDevice(t *testing.T) {
	next := splice(t, fragmentConfig, "api-router",
		"name: api-router\ntype: router\n# preserved router edit\n")

	if !strings.Contains(next, "    # preserved router edit") {
		t.Errorf("comment inside the edited device was dropped:\n%s", next)
	}
}

func TestSpliceDeviceFragmentIsIdentityForAnUnchangedBlock(t *testing.T) {
	fragment, ok := config.FindDeviceFragment(fragmentConfig, "api-router")
	if !ok {
		t.Fatal("no fragment")
	}

	body := fragmentConfig[fragment.Start:fragment.End]
	dedented := strings.ReplaceAll(strings.TrimPrefix(strings.TrimSuffix(body, "\n"), "  - "), "\n    ", "\n")

	if next := config.SpliceDeviceFragment(fragmentConfig, fragment, dedented); next != fragmentConfig {
		t.Errorf("round trip changed the document:\ngot:\n%s\nwant:\n%s", next, fragmentConfig)
	}
}

// Adjacent items are how the daemon writes every config it saves, since
// yaml.Marshal puts no blank line between sequence items. Splicing the first
// must not swallow the second.
func TestSpliceDeviceFragmentLeavesAnAdjacentDeviceIntact(t *testing.T) {
	adjacent := "devices:\n  - name: dev-a\n    type: switch\n  - name: dev-b\n    type: server\n"

	next := splice(t, adjacent, "dev-a", "name: dev-a\ntype: firewall\n")

	want := "devices:\n  - name: dev-a\n    type: firewall\n  - name: dev-b\n    type: server\n"
	if next != want {
		t.Errorf("got:\n%s\nwant:\n%s", next, want)
	}
}

func TestFindDeviceFragmentHandlesTheLastDeviceAndTrailingComments(t *testing.T) {
	trailing := "devices:\n  - name: only\n    type: switch\n# a trailing note\n"

	next := splice(t, trailing, "only", "name: only\ntype: router\n")

	if !strings.Contains(next, "# a trailing note") {
		t.Errorf("a comment after the last device was swallowed:\n%s", next)
	}

	if !strings.Contains(next, "    type: router") {
		t.Errorf("edit did not land:\n%s", next)
	}
}

func TestFindDeviceFragmentRejectsWhatItCannotLocate(t *testing.T) {
	for name, source := range map[string]string{
		"unparseable":               "devices: [\n",
		"no devices key":            "networks:\n  - name: lan\n",
		"devices is not a sequence": "devices:\n  name: nope\n",
	} {
		if _, ok := config.FindDeviceFragment(source, "api-router"); ok {
			t.Errorf("%s: located a fragment it should not have", name)
		}
	}

	if _, ok := config.FindDeviceFragment(fragmentConfig, "absent"); ok {
		t.Error("located a device that is not in the config")
	}
}
