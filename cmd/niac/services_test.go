package main

import (
	"strings"
	"testing"
)

func TestDefaultStoragePath(t *testing.T) {
	path := defaultStoragePath()
	if path == "" {
		t.Error("defaultStoragePath() returned empty string")
	}
	if !strings.HasSuffix(path, "niac.db") {
		t.Errorf("defaultStoragePath() = %q, should end with niac.db", path)
	}
}

func TestResolveServiceDefaults(t *testing.T) {
	t.Run("nil services", func(_ *testing.T) {
		// Should not panic
		resolveServiceDefaults(nil)
	})

	t.Run("empty storage path", func(t *testing.T) {
		services := &serviceOptions{}
		resolveServiceDefaults(services)
		if services.storagePath == "" {
			t.Error("Expected storagePath to be set after resolveServiceDefaults")
		}
	})

	t.Run("existing storage path preserved", func(t *testing.T) {
		services := &serviceOptions{storagePath: "/custom/path.db"}
		resolveServiceDefaults(services)
		if services.storagePath != "/custom/path.db" {
			t.Errorf("Expected storagePath to be preserved, got %q", services.storagePath)
		}
	})
}

func TestServiceOptionsStruct(t *testing.T) {
	opts := &serviceOptions{
		storagePath: "/data/niac.db",
	}

	if opts.storagePath != "/data/niac.db" {
		t.Errorf("storagePath = %q, want %q", opts.storagePath, "/data/niac.db")
	}
}
