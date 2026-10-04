package capacity

import (
	"os"

	"golang.org/x/sys/unix"
)

func totalMemory() (uint64, error) {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return 0, err
	}
	physical := info.Totalram * uint64(info.Unit)

	// sysinfo(2) reports the host even inside a container. /proc/meminfo is
	// rewritten by lxcfs in an LXC container, and the cgroup files carry a
	// Docker- or systemd-style limit; the tightest of them wins.
	if raw, err := os.ReadFile("/proc/meminfo"); err == nil {
		if total, ok := parseMemTotal(string(raw)); ok && total < physical {
			physical = total
		}
	}
	limitFiles := []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	}
	contents := make([]string, 0, len(limitFiles))
	for _, path := range limitFiles {
		if raw, err := os.ReadFile(path); err == nil {
			contents = append(contents, string(raw))
		}
	}
	return effectiveMemory(physical, contents), nil
}
