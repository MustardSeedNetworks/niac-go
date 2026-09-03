package version

import "runtime/debug"

// ExtractVersionFromBuildInfo exposes extractVersionFromBuildInfo to the
// external test package.
func ExtractVersionFromBuildInfo(info *debug.BuildInfo) (string, string, string) {
	return extractVersionFromBuildInfo(info)
}
