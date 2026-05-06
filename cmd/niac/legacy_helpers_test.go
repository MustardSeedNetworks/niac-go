package main

import (
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func TestApplyLegacyServiceFlags(t *testing.T) {
	t.Run("all flags set", testLegacyFlagsAllSet)
	t.Run("no flags set uses defaults", testLegacyFlagsNone)
	t.Run("partial flags set", testLegacyFlagsPartial)
	t.Run("does not override existing storagePath", testLegacyFlagsKeepExisting)
	t.Run("flag storagePath overrides existing", testLegacyFlagsOverrideExisting)
	t.Run("zero threshold not applied", testLegacyFlagsZeroThreshold)
}

func testLegacyFlagsAllSet(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	flags.apiListen = ":8080"
	flags.apiToken = "secret"
	flags.metricsListen = ":9090"
	flags.storagePath = "/custom/niac.db"
	flags.alertPacketsThreshold = 5000
	flags.alertWebhook = "https://hooks.example.com/alert"

	services := &serviceOptions{}
	applyLegacyServiceFlags(flags, services)

	assertStrEq(t, "apiListen", services.apiListen, ":8080")
	assertStrEq(t, "apiToken", services.apiToken, "secret")
	assertStrEq(t, "metricsListen", services.metricsListen, ":9090")
	assertStrEq(t, "storagePath", services.storagePath, "/custom/niac.db")
	if services.alertPacketsThreshold != 5000 {
		t.Errorf("alertPacketsThreshold = %d, want %d", services.alertPacketsThreshold, 5000)
	}
	assertStrEq(t, "alertWebhook", services.alertWebhook, "https://hooks.example.com/alert")
}

func testLegacyFlagsNone(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	services := &serviceOptions{}
	applyLegacyServiceFlags(flags, services)

	if services.apiListen != "" {
		t.Errorf("apiListen should be empty, got %q", services.apiListen)
	}
	if services.storagePath == "" {
		t.Error("Expected storagePath to have a default value")
	}
}

func testLegacyFlagsPartial(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	flags.apiListen = ":8080"
	services := &serviceOptions{}
	applyLegacyServiceFlags(flags, services)

	assertStrEq(t, "apiListen", services.apiListen, ":8080")
	if services.apiToken != "" {
		t.Errorf("apiToken should be empty, got %q", services.apiToken)
	}
	if services.storagePath == "" {
		t.Error("Expected storagePath to have default")
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

func testLegacyFlagsZeroThreshold(t *testing.T) {
	t.Helper()
	flags := newLegacyFlags()
	flags.alertPacketsThreshold = 0
	services := &serviceOptions{alertPacketsThreshold: 100}
	applyLegacyServiceFlags(flags, services)

	if services.alertPacketsThreshold != 100 {
		t.Errorf("alertPacketsThreshold should be preserved when flag is 0, got %d",
			services.alertPacketsThreshold)
	}
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
