package capacity

import "testing"

func TestDevicesFor(t *testing.T) {
	t.Parallel()

	const gib = uint64(1) << 30
	tests := []struct {
		name   string
		memory uint64
		want   int
	}{
		{name: "tiny host keeps the minimum", memory: 256 << 20, want: minDevices},
		{name: "at the minimum boundary", memory: reservedBytes + minDevices*bytesPerDevice, want: minDevices},
		{name: "1 GiB", memory: gib, want: 768},
		{name: "2 GiB lab container runs all six presentation packs", memory: 2 * gib, want: 1792},
		{name: "16 GiB", memory: 16 * gib, want: 16128},
		{name: "256 GiB", memory: 256 * gib, want: 261888},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := devicesFor(tc.memory); got != tc.want {
				t.Fatalf("devicesFor(%d) = %d, want %d", tc.memory, got, tc.want)
			}
		})
	}
}

func TestPresentationPacksFitTheReferenceLabHost(t *testing.T) {
	t.Parallel()
	// #2469: the six resized packs (hospital 257, warehouse 265 and four more of
	// about 250) need about 1,550 devices together; the lab container has 2 GiB.
	const presentationPacks = 1550
	if got := devicesFor(2 << 30); got < presentationPacks {
		t.Fatalf("a 2 GiB host carries %d devices, want at least %d", got, presentationPacks)
	}
}

func TestEffectiveMemory(t *testing.T) {
	t.Parallel()

	const physical = uint64(16) << 30
	tests := []struct {
		name    string
		cgroups []string
		want    uint64
	}{
		{name: "no cgroup files", cgroups: nil, want: physical},
		{name: "v2 unlimited", cgroups: []string{"max\n"}, want: physical},
		{name: "v2 limit applies", cgroups: []string{"2147483648\n"}, want: 2 << 30},
		{name: "v1 unlimited is larger than physical", cgroups: []string{"9223372036854771712\n"}, want: physical},
		{name: "tightest of v2 and v1 wins", cgroups: []string{"4294967296", "2147483648"}, want: 2 << 30},
		{name: "unparseable content is ignored", cgroups: []string{"garbage", ""}, want: physical},
		{name: "zero is not a limit", cgroups: []string{"0"}, want: physical},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := effectiveMemory(physical, tc.cgroups); got != tc.want {
				t.Fatalf("effectiveMemory() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestMaxDevicesReadsThisHost(t *testing.T) {
	t.Parallel()
	got := MaxDevices()
	if got < minDevices {
		t.Fatalf("MaxDevices() = %d, want at least %d", got, minDevices)
	}
	t.Logf("this host's device budget: %d", got)
}

func TestParseMemTotal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		meminfo string
		want    uint64
		wantOK  bool
	}{
		{
			// CT304 under lxcfs: the container's 6 GiB limit, while sysinfo(2)
			// in the same container reports the host's 62 GiB.
			name:    "lxcfs container limit",
			meminfo: "MemTotal:        6291456 kB\nMemFree:         5442560 kB\n",
			want:    6 << 30, wantOK: true,
		},
		{name: "not the first line", meminfo: "Foo: 1 kB\nMemTotal: 1024 kB\n", want: 1 << 20, wantOK: true},
		{name: "missing", meminfo: "MemFree: 1 kB\n", wantOK: false},
		{name: "garbage value", meminfo: "MemTotal: lots kB\n", wantOK: false},
		{name: "empty", meminfo: "", wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseMemTotal(tc.meminfo)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("parseMemTotal() = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
