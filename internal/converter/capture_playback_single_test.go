package converter

import (
	"strings"
	"testing"
)

// The runtime has always played back exactly one capture: the config carries a
// single CapturePlayback, the replay controller builds one engine. The YAML
// schema nonetheless accepted a list, kept the first entry and dropped the rest
// - and worse, a load followed by a save wrote the survivor back out as a
// one-element list, so a round trip deleted the others from disk.
//
// Nothing ships more than one, so this is refused at the schema rather than
// answered with multi-playback support nobody asked for.
func TestMoreThanOneCapturePlaybackIsRefused(t *testing.T) {
	cfg := &Config{
		CapturePlaybacks: []CapturePlayback{
			{FileName: "first.pcap"},
			{FileName: "second.pcap"},
		},
		Devices: []Device{validPlaybackDevice()},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("a config declaring two capture playbacks was accepted")
	}
	if !strings.Contains(err.Error(), "capture_playbacks") {
		t.Errorf("error = %v, want it to name the offending field", err)
	}
	// An operator reads this, so it has to say what is wrong rather than name
	// the rule that caught it.
	if !strings.Contains(err.Error(), "at most 1") {
		t.Errorf("error = %v, want it to state the limit", err)
	}
}

// One playback is what the runtime plays, and it stays valid.
func TestOneCapturePlaybackIsAccepted(t *testing.T) {
	cfg := &Config{
		CapturePlaybacks: []CapturePlayback{{FileName: "only.pcap"}},
		Devices:          []Device{validPlaybackDevice()},
	}

	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("ValidateConfig: %v", err)
	}
}

func validPlaybackDevice() Device {
	return Device{
		Name: "SW01", Type: "switch",
		MAC: "00:11:22:33:44:55", IPs: []string{"10.0.0.1"},
	}
}
