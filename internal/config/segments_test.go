package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// TestSegmentConfigPathResolution: a segment's `config:` file (a whole demo) is
// loaded and its devices become the segment's device set — how two real demo
// configs are run side-by-side.
func TestSegmentConfigPathResolution(t *testing.T) {
	dir := t.TempDir()

	demo1 := filepath.Join(dir, "demo1.yaml")
	if err := os.WriteFile(demo1, []byte(`
devices:
  - name: demo1-r1
    mac: "02:00:00:00:02:01"
    ips: ["10.0.0.1"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	parent := filepath.Join(dir, "multi.yaml")
	if err := os.WriteFile(parent, []byte(`
segments:
  - tag: 200
    config: demo1.yaml
  - tag: 300
    devices:
      - name: demo2-r1
        mac: "02:00:00:00:03:01"
        ips: ["10.0.0.1"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadYAML(parent)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	segs := cfg.NormalizedSegments()
	if len(segs) != 2 {
		t.Fatalf("segments = %d, want 2", len(segs))
	}

	if segs[0].Tag != 200 || len(segs[0].Devices) != 1 || segs[0].Devices[0].Name != "demo1-r1" {
		t.Errorf("tag-200 segment should resolve demo1.yaml to demo1-r1, got %+v", segs[0])
	}

	if segs[1].Tag != 300 || segs[1].Devices[0].Name != "demo2-r1" {
		t.Errorf("tag-300 inline segment wrong: %+v", segs[1])
	}

	data, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalConfigYAML() error = %v", err)
	}
	var authored converter.Config
	if err = yaml.Unmarshal(data, &authored); err != nil {
		t.Fatalf("unmarshal serialized config: %v", err)
	}
	if authored.Segments[0].Config != "" || len(authored.Segments[0].Devices) != 1 {
		t.Fatalf("resolved segment serialized as unresolved reference: %#v", authored.Segments[0])
	}
}

func TestLoadYAMLManagedConfinesSegmentConfigs(t *testing.T) {
	base := t.TempDir()
	managedRoot := filepath.Join(base, "networks")
	if err := os.Mkdir(managedRoot, 0o750); err != nil {
		t.Fatal(err)
	}

	childYAML := []byte(`
devices:
  - name: child
    mac: "02:00:00:00:02:01"
    ips: ["10.0.0.1"]
`)
	managedChild := filepath.Join(managedRoot, "child.yaml")
	if err := os.WriteFile(managedChild, childYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	outsideChild := filepath.Join(base, "outside.yaml")
	if err := os.WriteFile(outsideChild, childYAML, 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkChild := filepath.Join(managedRoot, "escape.yaml")
	if err := os.Symlink(outsideChild, symlinkChild); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		childPath string
		wantErr   bool
	}{
		{name: "managed child", childPath: managedChild},
		{name: "absolute escape", childPath: outsideChild, wantErr: true},
		{name: "relative escape", childPath: "../outside.yaml", wantErr: true},
		{name: "symlink escape", childPath: symlinkChild, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := filepath.Join(managedRoot, tt.name+".yaml")
			parentYAML := fmt.Sprintf(`
segments:
  - tag: 200
    config: %q
`, tt.childPath)
			if err := os.WriteFile(parent, []byte(parentYAML), 0o600); err != nil {
				t.Fatal(err)
			}

			_, _, err := LoadYAMLManaged(parent, []string{managedRoot})
			if tt.wantErr {
				if !errors.Is(err, ErrPathOutsideManagedRoots) {
					t.Fatalf("LoadYAMLManaged() error = %v, want ErrPathOutsideManagedRoots", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadYAMLManaged() error = %v", err)
			}
		})
	}
}

// TestNormalizedSegmentsBackwardCompat: a bare `devices:` config presents as a
// single untagged segment — today's flat behavior, unchanged.
func TestNormalizedSegmentsBackwardCompat(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(`
devices:
  - name: r1
    mac: "02:00:00:00:00:01"
    ips: ["10.0.0.1"]
`))
	if err != nil {
		t.Fatalf("LoadYAMLBytes: %v", err)
	}

	if len(cfg.Segments) != 0 {
		t.Errorf("flat config should have no explicit segments, got %d", len(cfg.Segments))
	}

	segs := cfg.NormalizedSegments()
	if len(segs) != 1 {
		t.Fatalf("NormalizedSegments = %d, want 1", len(segs))
	}

	if segs[0].Tag != UntaggedTag {
		t.Errorf("implicit segment tag = %d, want untagged (%d)", segs[0].Tag, UntaggedTag)
	}

	if len(segs[0].Devices) != 1 || segs[0].Devices[0].Name != "r1" {
		t.Errorf("implicit segment should carry the flat device list")
	}
}

// TestSegmentsParse: explicit segments bind device sets to tags; both `untagged`
// and a bare integer tag are accepted.
func TestSegmentsParse(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(`
segments:
  - tag: untagged
    devices:
      - name: mgmt
        mac: "02:00:00:00:00:0a"
        ips: ["10.0.0.1"]
  - tag: 200
    devices:
      - name: evt
        mac: "02:00:00:00:00:14"
        ips: ["10.20.200.1"]
`))
	if err != nil {
		t.Fatalf("LoadYAMLBytes: %v", err)
	}

	segs := cfg.NormalizedSegments()
	if len(segs) != 2 {
		t.Fatalf("segments = %d, want 2", len(segs))
	}

	if segs[0].Tag != UntaggedTag {
		t.Errorf("segment 0 tag = %d, want untagged", segs[0].Tag)
	}

	if segs[1].Tag != 200 {
		t.Errorf("segment 1 tag = %d, want 200", segs[1].Tag)
	}

	if segs[1].Devices[0].Name != "evt" {
		t.Errorf("segment 1 device = %q, want evt", segs[1].Devices[0].Name)
	}
}

func TestSegmentsRejectDuplicateNormalizedTags(t *testing.T) {
	for _, test := range []struct {
		name      string
		firstTag  string
		secondTag string
	}{
		{name: "numeric", firstTag: "200", secondTag: "\"200\""},
		{name: "untagged", firstTag: "untagged", secondTag: "UNTAGGED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			yamlConfig := fmt.Sprintf(`
segments:
  - tag: %s
    devices:
      - name: first
        mac: "02:00:00:00:00:01"
        ips: ["10.0.0.1"]
  - tag: %s
    devices:
      - name: second
        mac: "02:00:00:00:00:02"
        ips: ["10.0.0.2"]
`, test.firstTag, test.secondTag)
			_, err := LoadYAMLBytes([]byte(yamlConfig))
			if !errors.Is(err, ErrDuplicateSegmentTag) {
				t.Fatalf("LoadYAMLBytes() error = %v, want duplicate segment tag", err)
			}
			for _, location := range []string{"segments[0].tag", "segments[1].tag"} {
				if !strings.Contains(err.Error(), location) {
					t.Fatalf("LoadYAMLBytes() error = %q, want location %q", err, location)
				}
			}
		})
	}
}

func TestDeviceCountIncludesSegments(t *testing.T) {
	flat, _ := LoadYAMLBytes([]byte(`
devices:
  - name: a
    mac: "02:00:00:00:00:01"
    ips: ["10.0.0.1"]
`))
	if flat.DeviceCount() != 1 {
		t.Errorf("flat DeviceCount = %d, want 1", flat.DeviceCount())
	}

	seg, _ := LoadYAMLBytes([]byte(`
segments:
  - tag: 200
    devices:
      - name: a
        mac: "02:00:00:00:00:01"
        ips: ["10.0.0.1"]
  - tag: 300
    devices:
      - name: b
        mac: "02:00:00:00:00:02"
        ips: ["10.0.0.2"]
`))
	if seg.DeviceCount() != 2 {
		t.Errorf("segmented DeviceCount = %d, want 2 (must count devices inside segments)", seg.DeviceCount())
	}
}

func TestSegmentsRejectMixedWithTopLevelDevices(t *testing.T) {
	_, err := LoadYAMLBytes([]byte(`
devices:
  - name: r1
    mac: "02:00:00:00:00:01"
    ips: ["10.0.0.1"]
segments:
  - tag: 200
    devices:
      - name: evt
        mac: "02:00:00:00:00:14"
        ips: ["10.20.200.1"]
`))
	if !errors.Is(err, ErrSegmentsAndTopLevelDevices) {
		t.Errorf("mixing top-level devices and segments should fail with ErrSegmentsAndTopLevelDevices, got %v", err)
	}
}

func TestSegmentTagValidation(t *testing.T) {
	for _, bad := range []string{"0", "4095", "abc", "-1"} {
		_, err := LoadYAMLBytes([]byte(`
segments:
  - tag: "` + bad + `"
    devices:
      - name: d
        mac: "02:00:00:00:00:01"
        ips: ["10.0.0.1"]
`))
		if !errors.Is(err, ErrInvalidSegmentTag) {
			t.Errorf("tag %q should be rejected with ErrInvalidSegmentTag, got %v", bad, err)
		}
	}
}

func TestSegmentDevicesXORConfig(t *testing.T) {
	// Neither devices nor config.
	_, err := LoadYAMLBytes([]byte(`
segments:
  - tag: 200
`))
	if !errors.Is(err, ErrSegmentDevicesXORConfig) {
		t.Errorf("segment with neither devices nor config should fail, got %v", err)
	}

	// Both devices and config.
	_, err = LoadYAMLBytes([]byte(`
segments:
  - tag: 200
    config: other.yaml
    devices:
      - name: d
        mac: "02:00:00:00:00:01"
        ips: ["10.0.0.1"]
`))
	if !errors.Is(err, ErrSegmentDevicesXORConfig) {
		t.Errorf("segment with both devices and config should fail, got %v", err)
	}
}
