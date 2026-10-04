package capacity

import (
	"strconv"
	"strings"
)

// parseCgroupLimit reads a cgroup memory limit file's contents: cgroup v2's
// memory.max ("max" when unlimited) or cgroup v1's memory.limit_in_bytes (a
// very large number when unlimited, which the caller's comparison against
// physical memory discards). The bool is false when the file sets no limit.
func parseCgroupLimit(content string) (uint64, bool) {
	value := strings.TrimSpace(content)
	if value == "" || value == "max" {
		return 0, false
	}
	limit, err := strconv.ParseUint(value, 10, 64)
	if err != nil || limit == 0 {
		return 0, false
	}
	return limit, true
}

// effectiveMemory is the smaller of physical memory and the tightest cgroup
// limit that applies, so a container is sized by what it may use.
func effectiveMemory(physical uint64, cgroupFiles []string) uint64 {
	memory := physical
	for _, content := range cgroupFiles {
		if limit, ok := parseCgroupLimit(content); ok && limit < memory {
			memory = limit
		}
	}
	return memory
}
