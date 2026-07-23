package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func defaultCertDir() string {
	if dir := os.Getenv("NIAC_CERT_DIR"); dir != "" {
		return dir
	}
	if runtime.GOOS != "windows" {
		return ""
	}

	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "NiAC", "certs")
}
