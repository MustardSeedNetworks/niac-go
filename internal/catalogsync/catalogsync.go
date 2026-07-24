package catalogsync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

const ManifestName = "catalog-source.json"

const (
	directoryMode               = 0o750
	fileMode                    = 0o600
	executableMode              = 0o700
	maxReportedValidationErrors = 20
	maxValidationErrorLines     = 3
)

type Mode string

const (
	ModeSync  Mode = "sync"
	ModeCheck Mode = "check"
)

type Options struct {
	Mode        Mode
	CatalogDir  string
	ExamplesDir string
	Repository  string
	Commit      string
}

type manifest struct {
	SchemaVersion int            `json:"schema_version"`
	Repository    string         `json:"repository"`
	Commit        string         `json:"commit"`
	Files         []manifestFile `json:"files"`
}

type manifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type mapping struct {
	source      string
	destination string
	optional    bool
}

type stagedWalk struct {
	path   string
	strict bool
}

var commitPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}([0-9a-fA-F]{24})?$`)

func Run(options Options) error {
	if err := validateOptions(options); err != nil {
		return err
	}

	examplesDir, resolveErr := filepath.Abs(options.ExamplesDir)
	if resolveErr != nil {
		return fmt.Errorf("resolve examples directory: %w", resolveErr)
	}
	if filepath.Dir(examplesDir) == examplesDir {
		return errors.New("examples directory cannot be a filesystem root")
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(examplesDir), directoryMode); mkdirErr != nil {
		return fmt.Errorf("create examples parent: %w", mkdirErr)
	}

	stage, stageErr := os.MkdirTemp(filepath.Dir(examplesDir), ".niac-catalog-stage-")
	if stageErr != nil {
		return fmt.Errorf("create catalog stage: %w", stageErr)
	}
	defer func() { _ = os.RemoveAll(stage) }()

	if populateErr := populateStage(options.CatalogDir, stage); populateErr != nil {
		return populateErr
	}
	if validationErr := validateStage(stage); validationErr != nil {
		return validationErr
	}
	if manifestErr := writeManifest(stage, options.Repository, strings.ToLower(options.Commit)); manifestErr != nil {
		return manifestErr
	}

	switch options.Mode {
	case ModeSync:
		return replaceTree(stage, examplesDir)
	case ModeCheck:
		return compareTrees(stage, examplesDir)
	default:
		return fmt.Errorf("unsupported catalog mode %q", options.Mode)
	}
}

func validateOptions(options Options) error {
	if options.Mode != ModeSync && options.Mode != ModeCheck {
		return fmt.Errorf("mode must be %q or %q", ModeSync, ModeCheck)
	}
	if options.CatalogDir == "" || options.ExamplesDir == "" {
		return errors.New("catalog and examples directories are required")
	}
	if strings.TrimSpace(options.Repository) == "" {
		return errors.New("catalog repository is required")
	}
	if !commitPattern.MatchString(options.Commit) {
		return errors.New("catalog commit must be a full 40- or 64-character hexadecimal revision")
	}
	return nil
}

func populateStage(catalogDir, stage string) error {
	for _, item := range mappings() {
		source := filepath.Join(catalogDir, filepath.FromSlash(item.source))
		destination := filepath.Join(stage, filepath.FromSlash(item.destination))
		info, statErr := os.Lstat(source)
		if statErr != nil {
			if item.optional && errors.Is(statErr, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("catalog source %s: %w", item.source, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("catalog source %s is a symbolic link", item.source)
		}
		if info.IsDir() {
			if copyErr := copyTree(source, destination); copyErr != nil {
				return fmt.Errorf("copy catalog source %s: %w", item.source, copyErr)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("catalog source %s is not a regular file", item.source)
		}
		if copyErr := copyFile(source, destination, info.Mode()); copyErr != nil {
			return fmt.Errorf("copy catalog source %s: %w", item.source, copyErr)
		}
	}
	return nil
}

func mappings() []mapping {
	return []mapping{
		{source: "scenarios/go-yaml"},
		{source: "walks/raw", destination: "device_walks"},
		{source: "walks/sanitized", destination: "device_walks_sanitized"},
		{source: "captures/shared", destination: "captures"},
		{source: "captures/go-extra", destination: "pcaps"},
		{source: "tools/walk-scripts/go", destination: "walk_scripts"},
		{
			source:      "tools/walk-scripts/java/run_demo.sh",
			destination: "walk_scripts/run_demo.sh",
			optional:    true,
		},
		{source: "docs/imported/go-examples"},
	}
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, relativeErr := filepath.Rel(source, path)
		if relativeErr != nil {
			return relativeErr
		}
		target := filepath.Join(destination, relative)
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link", path)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, directoryMode)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(source, destination string, mode fs.FileMode) error {
	if mkdirErr := os.MkdirAll(filepath.Dir(destination), directoryMode); mkdirErr != nil {
		return mkdirErr
	}
	input, openErr := os.Open(source)
	if openErr != nil {
		return openErr
	}
	defer func() { _ = input.Close() }()

	permissions := fs.FileMode(fileMode)
	if mode&0o111 != 0 {
		permissions = executableMode
	}
	output, createErr := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
	if createErr != nil {
		return createErr
	}
	if _, copyErr := io.Copy(output, input); copyErr != nil {
		_ = output.Close()
		return copyErr
	}
	return output.Close()
}

func validateStage(stage string) error {
	scenarios, walks, inspectErr := catalogValidationInputs(stage)
	if inspectErr != nil {
		return inspectErr
	}
	scenarioErrors := validateScenarios(stage, scenarios)
	if len(scenarioErrors) > 0 {
		return joinValidationErrors(scenarioErrors)
	}
	return joinValidationErrors(validateWalks(stage, walks))
}

func catalogValidationInputs(stage string) ([]string, []stagedWalk, error) {
	var scenarios []string
	var walks []stagedWalk
	inspectErr := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, relativeErr := filepath.Rel(stage, path)
		if relativeErr != nil {
			return relativeErr
		}
		switch {
		case strings.HasSuffix(relative, ".yaml"), strings.HasSuffix(relative, ".yml"):
			scenarios = append(scenarios, path)
		case within(relative, "device_walks_sanitized"):
			walks = append(walks, stagedWalk{path: path, strict: true})
		case within(relative, "device_walks"):
			walks = append(walks, stagedWalk{path: path})
		}
		return nil
	})
	if inspectErr != nil {
		return nil, nil, fmt.Errorf("inspect staged catalog: %w", inspectErr)
	}
	return scenarios, walks, nil
}

func validateScenarios(stage string, scenarios []string) []error {
	var validationErrors []error
	for _, path := range scenarios {
		if scenarioErr := validateScenario(path); scenarioErr != nil {
			relative, _ := filepath.Rel(stage, path)
			validationErrors = append(validationErrors, fmt.Errorf("%s: %s", relative, summarizeError(scenarioErr)))
		}
	}
	return validationErrors
}

func validateWalks(stage string, walks []stagedWalk) []error {
	var validationErrors []error
	for _, walk := range walks {
		result, validationErr := snmp.ValidateWalkFile(walk.path)
		invalid := validationErr != nil ||
			(walk.strict && !result.Valid) ||
			(!walk.strict && result.ValidLines == 0)
		if invalid {
			relative, _ := filepath.Rel(stage, walk.path)
			if validationErr != nil {
				validationErrors = append(
					validationErrors,
					fmt.Errorf("%s: %s", relative, summarizeError(validationErr)),
				)
			} else {
				validationErrors = append(validationErrors, fmt.Errorf("%s: walk validation failed", relative))
			}
		}
	}
	return validationErrors
}

func summarizeError(err error) string {
	lines := strings.SplitN(err.Error(), "\n", maxValidationErrorLines)
	if len(lines) == 1 {
		return lines[0]
	}
	return lines[0] + " " + strings.TrimSpace(lines[1])
}

func joinValidationErrors(validationErrors []error) error {
	if len(validationErrors) <= maxReportedValidationErrors {
		return errors.Join(validationErrors...)
	}
	reported := slices.Clone(validationErrors[:maxReportedValidationErrors])
	reported = append(
		reported,
		fmt.Errorf("%d additional validation errors omitted", len(validationErrors)-maxReportedValidationErrors),
	)
	return errors.Join(reported...)
}

func validateScenario(path string) error {
	cfg, loadErr := config.Load(path)
	if loadErr != nil {
		return fmt.Errorf("load scenario: %w", loadErr)
	}
	result := config.NewValidator(path).Validate(cfg)
	if !result.Valid {
		return fmt.Errorf("scenario validation failed: %s", result.Format())
	}
	for _, attachment := range cfg.Attachments {
		report := fabric.Compile(cfg, fabric.Binding{
			Mode:           fabric.ModeDirect,
			Attachment:     attachment.Name,
			PolicyApproved: true,
		})
		if !report.Safe {
			return fmt.Errorf(
				"routed scenario validation failed for attachment %q: %v",
				attachment.Name,
				report.Diagnostics,
			)
		}
	}
	return nil
}

func within(path, directory string) bool {
	path = filepath.Clean(path)
	directory = filepath.Clean(directory)
	return path == directory || strings.HasPrefix(path, directory+string(filepath.Separator))
}

func writeManifest(stage, repository, commit string) error {
	files, filesErr := treeFiles(stage)
	if filesErr != nil {
		return filesErr
	}
	document := manifest{
		SchemaVersion: 1,
		Repository:    repository,
		Commit:        commit,
		Files:         files,
	}
	data, marshalErr := json.MarshalIndent(document, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("encode source manifest: %w", marshalErr)
	}
	data = append(data, '\n')
	if writeErr := os.WriteFile(filepath.Join(stage, ManifestName), data, fileMode); writeErr != nil {
		return fmt.Errorf("write source manifest: %w", writeErr)
	}
	return nil
}

func treeFiles(root string) ([]manifestFile, error) {
	var files []manifestFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, relativeErr := filepath.Rel(root, path)
		if relativeErr != nil {
			return relativeErr
		}
		if relative == ManifestName {
			return nil
		}
		sum, hashErr := fileSHA256(path)
		if hashErr != nil {
			return hashErr
		}
		files = append(files, manifestFile{
			Path:   filepath.ToSlash(relative),
			SHA256: sum,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("build source manifest: %w", err)
	}
	slices.SortFunc(files, func(left, right manifestFile) int {
		return strings.Compare(left.Path, right.Path)
	})
	return files, nil
}

func fileSHA256(path string) (string, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return "", openErr
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", copyErr
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func compareTrees(expected, actual string) error {
	expectedFiles, expectedErr := allTreeFiles(expected)
	if expectedErr != nil {
		return fmt.Errorf("read staged catalog: %w", expectedErr)
	}
	actualFiles, actualErr := allTreeFiles(actual)
	if actualErr != nil {
		return fmt.Errorf("read generated examples: %w", actualErr)
	}
	if !slices.Equal(expectedFiles, actualFiles) {
		return errors.New("generated examples file set does not match the catalog")
	}
	for _, relative := range expectedFiles {
		expectedHash, expectedHashErr := fileSHA256(filepath.Join(expected, relative))
		if expectedHashErr != nil {
			return expectedHashErr
		}
		actualHash, actualHashErr := fileSHA256(filepath.Join(actual, relative))
		if actualHashErr != nil {
			return actualHashErr
		}
		if expectedHash != actualHash {
			return fmt.Errorf("generated example differs from catalog: %s", filepath.ToSlash(relative))
		}
	}
	return nil
}

func allTreeFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, relativeErr := filepath.Rel(root, path)
		if relativeErr != nil {
			return relativeErr
		}
		files = append(files, relative)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

func replaceTree(stage, destination string) error {
	backup, backupErr := os.MkdirTemp(filepath.Dir(destination), ".niac-catalog-backup-")
	if backupErr != nil {
		return fmt.Errorf("reserve examples backup: %w", backupErr)
	}
	if removeErr := os.Remove(backup); removeErr != nil {
		return fmt.Errorf("prepare examples backup: %w", removeErr)
	}
	defer func() { _ = os.RemoveAll(backup) }()

	if _, statErr := os.Stat(destination); statErr == nil {
		if renameErr := os.Rename(destination, backup); renameErr != nil {
			return fmt.Errorf("preserve existing examples: %w", renameErr)
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("inspect existing examples: %w", statErr)
	}
	if renameErr := os.Rename(stage, destination); renameErr != nil {
		_ = os.Rename(backup, destination)
		return fmt.Errorf("install generated examples: %w", renameErr)
	}
	return nil
}
