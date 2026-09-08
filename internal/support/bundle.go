package support

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/version"
)

// BundleOptions names what a support bundle collects. Everything is optional:
// a bundle taken on a host where the daemon has never run still carries the
// manifest, which is the part support asks for first.
type BundleOptions struct {
	// Configs are scenario files to include. Each is loaded, stripped of its
	// credentials and written back as YAML.
	Configs []string

	// LogPath is the daemon log to include. Only its tail is taken.
	LogPath string

	// CertDir is the daemon's certificate directory. Certificates are listed
	// by name and fingerprint-free metadata only; no file in it is read into
	// the bundle, because it holds the private key.
	CertDir string

	// Token is the daemon's bearer token, if the caller holds it. It is never
	// written to the bundle -- it is passed so log lines quoting it can be
	// scrubbed by value.
	Token string

	// Interfaces is the host's interface inventory as the operator would see
	// it, one entry per line.
	Interfaces []string
}

// bundleLogTailBytes is how much of the log the bundle carries. A support
// bundle is mailed, and the last few megabytes hold the failure; the whole
// file makes it unsendable and adds nothing.
const bundleLogTailBytes = 4 << 20

// manifest is the bundle's first file: what was running, on what, when.
type manifest struct {
	CollectedAt  string            `json:"collectedAt"`
	Version      map[string]string `json:"version"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	GoVersion    string            `json:"goVersion"`
	Configs      []string          `json:"configs"`
	Certificates []string          `json:"certificates,omitempty"`
}

// WriteBundle collects diagnostics into a gzipped tar on w.
//
// Nothing reaches the archive without passing the redactor: configurations are
// stripped structurally over their typed credential fields, and the log tail
// is scrubbed of those same values plus anything else shaped like a
// credential. Private key material is never read at all.
func WriteBundle(options BundleOptions, w io.Writer) error {
	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("open gzip writer: %w", err)
	}
	tw := tar.NewWriter(gz)

	if collectErr := collect(tw, options); collectErr != nil {
		return collectErr
	}
	if closeErr := tw.Close(); closeErr != nil {
		return fmt.Errorf("close tar: %w", closeErr)
	}
	if closeErr := gz.Close(); closeErr != nil {
		return fmt.Errorf("close gzip: %w", closeErr)
	}
	return nil
}

// collect writes every file the bundle carries into tw. Configurations are
// read first because the credentials they declare are what the log scrub then
// removes from free text.
func collect(tw *tar.Writer, options BundleOptions) error {
	summary := manifest{
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Version:     version.Info(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
	}
	secrets := []string{options.Token}

	for _, source := range options.Configs {
		scenario, redactErr := redactConfigFile(source)
		if redactErr != nil {
			return redactErr
		}
		secrets = append(secrets, scenario.secrets...)
		summary.Configs = append(summary.Configs, scenario.name)
		if writeErr := writeBundleFile(tw, path.Join("configs", scenario.name), scenario.content); writeErr != nil {
			return writeErr
		}
	}

	certificates, listErr := listCertificates(options.CertDir)
	if listErr != nil {
		return listErr
	}
	summary.Certificates = certificates

	if manifestErr := writeManifest(tw, summary); manifestErr != nil {
		return manifestErr
	}
	if len(options.Interfaces) > 0 {
		inventory := strings.Join(options.Interfaces, "\n") + "\n"
		if writeErr := writeBundleFile(tw, "interfaces.txt", []byte(inventory)); writeErr != nil {
			return writeErr
		}
	}
	return writeLogTail(tw, options.LogPath, secrets)
}

// redactedConfig is one scenario as the bundle carries it: the credential-free
// document, and the credential values it used to hold so free text elsewhere
// can be scrubbed of the same strings.
type redactedConfig struct {
	name    string
	content []byte
	secrets []string
}

// redactConfigFile loads one scenario, removes its credentials and renders it
// back through the canonical YAML writer, so the bundle carries the same
// document the loader would read rather than a text-mangled copy.
func redactConfigFile(source string) (redactedConfig, error) {
	cfg, loadErr := config.Load(source)
	if loadErr != nil {
		return redactedConfig{}, fmt.Errorf("load %s: %w", source, loadErr)
	}
	secrets := Secrets(cfg)
	Redact(cfg)

	content, marshalErr := config.MarshalConfigYAML(cfg)
	if marshalErr != nil {
		return redactedConfig{}, fmt.Errorf("render %s: %w", source, marshalErr)
	}
	return redactedConfig{name: filepath.Base(source), content: content, secrets: secrets}, nil
}

// listCertificates records which certificates exist without reading any of
// them. The directory holds the daemon's private key, and a support bundle
// that carried it would hand support the host's identity.
func listCertificates(certDir string) ([]string, error) {
	if certDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(certDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", certDir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func writeManifest(tw *tar.Writer, summary manifest) error {
	encoded, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("render manifest: %w", err)
	}
	return writeBundleFile(tw, "manifest.json", append(encoded, '\n'))
}

func writeLogTail(tw *tar.Writer, logPath string, secrets []string) error {
	if logPath == "" {
		return nil
	}
	tail, readErr := readTail(logPath, bundleLogTailBytes)
	if os.IsNotExist(readErr) {
		return nil
	}
	if readErr != nil {
		return readErr
	}
	return writeBundleFile(tw, path.Join("logs", filepath.Base(logPath)), []byte(ScrubText(string(tail), secrets)))
}

// readTail returns the last limit bytes of path, starting at the first newline
// so the tail never opens mid-line.
func readTail(path string, limit int64) ([]byte, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, openErr
	}
	defer func() { _ = file.Close() }()

	info, statErr := file.Stat()
	if statErr != nil {
		return nil, fmt.Errorf("stat %s: %w", path, statErr)
	}
	truncated := info.Size() > limit
	if truncated {
		if _, seekErr := file.Seek(info.Size()-limit, io.SeekStart); seekErr != nil {
			return nil, fmt.Errorf("seek %s: %w", path, seekErr)
		}
	}
	data, readErr := io.ReadAll(file)
	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", path, readErr)
	}
	if truncated {
		if cut := bytes.IndexByte(data, '\n'); cut >= 0 {
			data = data[cut+1:]
		}
	}
	return data, nil
}

func writeBundleFile(tw *tar.Writer, name string, content []byte) error {
	header := &tar.Header{
		Name:     name,
		Mode:     archiveFileMode,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
		Format:   tar.FormatPAX,
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write header %s: %w", name, err)
	}
	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
