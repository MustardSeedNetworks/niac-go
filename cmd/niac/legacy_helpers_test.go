package main

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestApplyLegacyServiceFlags(t *testing.T) {
	t.Run("no flags set uses defaults", testLegacyFlagsNone)
	t.Run("does not override existing storagePath", testLegacyFlagsKeepExisting)
	t.Run("flag storagePath overrides existing", testLegacyFlagsOverrideExisting)
}

func testLegacyFlagsNone(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	services := &serviceOptions{}
	applyLegacyServiceFlags(flags, services)

	if services.storagePath == "" {
		t.Error("Expected storagePath to have a default value")
	}
}

func testLegacyFlagsKeepExisting(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	services := &serviceOptions{storagePath: "/existing/path.db"}
	applyLegacyServiceFlags(flags, services)

	assertStrEq(t, "storagePath", services.storagePath, "/existing/path.db")
}

func testLegacyFlagsOverrideExisting(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	flags.storagePath = "/new/path.db"
	services := &serviceOptions{storagePath: "/existing/path.db"}
	applyLegacyServiceFlags(flags, services)

	assertStrEq(t, "storagePath", services.storagePath, "/new/path.db")
}

func assertStrEq(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func TestLogCapturePlaybackDebug(t *testing.T) {
	t.Run("nil capture playback", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("logCapturePlaybackDebug panicked: %v", r)
			}
		}()

		cfg := &config.Config{}
		logCapturePlaybackDebug(cfg)
	})

	t.Run("with capture playback", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("logCapturePlaybackDebug panicked: %v", r)
			}
		}()

		cfg := &config.Config{
			CapturePlayback: &config.CapturePlayback{
				FileName:  "test.pcap",
				LoopTime:  1000,
				ScaleTime: 2.0,
			},
		}
		logCapturePlaybackDebug(cfg)
	})

	t.Run("with default scale time", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("logCapturePlaybackDebug panicked: %v", r)
			}
		}()

		cfg := &config.Config{
			CapturePlayback: &config.CapturePlayback{
				FileName:  "test.pcap",
				ScaleTime: 1.0,
			},
		}
		logCapturePlaybackDebug(cfg)
	})
}
