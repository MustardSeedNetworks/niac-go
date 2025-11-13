package main

import (
	"os"
	"path/filepath"
)

type serviceOptions struct {
	apiListen             string
	apiToken              string
	metricsListen         string
	storagePath           string
	alertPacketsThreshold uint64
	alertWebhook          string
}

var servicesOpts = serviceOptions{}

func defaultStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "niac", "niac.db")
	}
	return filepath.Join(home, ".niac", "niac.db")
}

func resolveServiceDefaults() {
	if servicesOpts.storagePath == "" {
		servicesOpts.storagePath = defaultStoragePath()
	}
}
