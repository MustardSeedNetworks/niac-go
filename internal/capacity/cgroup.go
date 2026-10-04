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

// parseMemTotal reads MemTotal from /proc/meminfo content, in bytes. Inside an
// LXC container lxcfs rewrites it to the container's memory limit, which is the
// only place that limit is visible: the container's own cgroup reads "max"
// because the limit sits on its parent, and sysinfo(2) reports the host.
func parseMemTotal(meminfo string) (uint64, bool) {
	for line := range strings.Lines(meminfo) {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kib, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil || kib == 0 {
				return 0, false
			}
			return kib * 1024, true
		}
	}
	return 0, false
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
