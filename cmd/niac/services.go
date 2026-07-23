package main

import (
	"os"
	"path/filepath"
)

type serviceOptions struct {
	storagePath string
}

func defaultStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "niac", "niac.db")
	}
	return filepath.Join(home, ".niac", "niac.db")
}

func resolveServiceDefaults(services *serviceOptions) {
	if services == nil {
		return
	}
	if services.storagePath == "" {
		services.storagePath = defaultStoragePath()
	}
}
