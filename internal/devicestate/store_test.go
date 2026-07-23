package devicestate_test

import (
	"errors"
	"net/netip"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestStoreUpdateIdentity(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})

	err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "edge-2"
		return identity, nil
	})
	if err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	snapshot := store.Snapshot()
	if snapshot.Identity.Hostname != "edge-2" {
		t.Fatalf("hostname = %q, want edge-2", snapshot.Identity.Hostname)
	}
	if snapshot.Version != 2 {
		t.Fatalf("version = %d, want 2", snapshot.Version)
	}
}

func TestStoreUpdateIdentityRollsBackOnError(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	wantErr := errors.New("reject change")

	err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "invalid"
		return identity, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("UpdateIdentity() error = %v, want %v", err, wantErr)
	}

	snapshot := store.Snapshot()
	if snapshot.Identity.Hostname != "edge-1" || snapshot.Version != 1 {
		t.Fatalf("snapshot = %#v, want unchanged initial state", snapshot)
	}
}

func TestStoreUpdateCallbackCanReadSnapshot(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		if got := store.Snapshot().Identity.Hostname; got != "edge-1" {
			t.Fatalf("hostname during callback = %q", got)
		}
		identity.Hostname = "edge-2"
		return identity, nil
	})
	if err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}
}

func TestStoreRejectsStaleCallbackCommit(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		store.SaveStartup()
		identity.Hostname = "stale"
		return identity, nil
	})
	if !errors.Is(err, devicestate.ErrConcurrentUpdate) {
		t.Fatalf("UpdateIdentity() error = %v, want ErrConcurrentUpdate", err)
	}
	if got := store.Snapshot().Identity.Hostname; got != "edge-1" {
		t.Fatalf("stale callback committed hostname %q", got)
	}
}

func TestStoreSerializesConcurrentUpdates(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	const updates = 20
	var waitGroup sync.WaitGroup

	for range updates {
		waitGroup.Go(func() {
			for {
				err := store.UpdateIdentity(identityNoop)
				if err == nil {
					return
				}
				if !errors.Is(err, devicestate.ErrConcurrentUpdate) {
					t.Errorf("UpdateIdentity() error = %v", err)
					return
				}
			}
		})
	}
	waitGroup.Wait()

	if got := store.Snapshot().Version; got != updates+1 {
		t.Fatalf("version = %d, want %d", got, updates+1)
	}
}

func identityNoop(identity devicestate.Identity) (devicestate.Identity, error) {
	return identity, nil
}

func TestStoreNetworkSnapshotIsImmutable(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{{
			Name: "Gi0/1", Address: netip.MustParsePrefix("10.0.0.1/24"), VLANs: []int{10},
		}},
		Routes: []devicestate.Route{{
			Destination: netip.MustParsePrefix("10.0.0.0/24"), Via: "Gi0/1", Connected: true,
		}},
	})

	snapshot := store.Snapshot()
	snapshot.Network.Interfaces[0].VLANs[0] = 20
	snapshot.Network.Interfaces[0].Name = "changed"

	got := store.Snapshot()
	if got.Network.Interfaces[0].Name != "Gi0/1" || got.Network.Interfaces[0].VLANs[0] != 10 {
		t.Fatalf("stored interface changed through snapshot: %#v", got.Network.Interfaces[0])
	}
	if got.Version != 2 {
		t.Fatalf("version = %d, want 2", got.Version)
	}
}
