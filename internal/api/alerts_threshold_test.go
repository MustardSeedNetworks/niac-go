package api

import (
	"testing"
	"time"
)

// One crossing is one alert, however many ticks it stays above the line.
//
// The rule used to be "fire whenever the total differs from the last one
// alerted". The packet total is cumulative, so on a running simulation it
// differs on every tick: one crossing sent a webhook every five seconds for
// the life of the session, to whatever endpoint the operator had configured.
//
// Asserted on the decision rather than on delivered POSTs because the send
// path refuses loopback by SSRF policy, so a local test receiver can never
// stand in for a real one.
func TestAlertFiresOncePerCrossingAcrossTenTicks(t *testing.T) {
	start := time.Now()
	firing := false
	lastAt := time.Time{}
	fired := 0

	for tick := range 10 {
		// A cumulative counter climbing past the threshold, as a live
		// simulation produces: every tick a different total.
		total := uint64(150 + tick*25)
		now := start.Add(time.Duration(tick) * alertTickerSecs * time.Second)

		fire, next := shouldFireAlert(firing, lastAt, total, 100, now)
		firing = next
		if fire {
			fired++
			lastAt = now
		}
	}

	if fired != 1 {
		t.Fatalf("alerts fired = %d over ten ticks above the threshold, want 1", fired)
	}
}

func TestAlertRenotifiesOnceTheCooldownExpires(t *testing.T) {
	start := time.Now()

	fire, firing := shouldFireAlert(false, time.Time{}, 150, 100, start)
	if !fire {
		t.Fatal("first crossing did not fire")
	}

	// Still across the threshold a moment later: silent.
	if quiet, _ := shouldFireAlert(firing, start, 200, 100, start.Add(time.Minute)); quiet {
		t.Fatal("fired again inside the cooldown")
	}

	// Still across it much later: worth saying again.
	if again, _ := shouldFireAlert(firing, start, 200, 100, start.Add(alertCooldown+time.Second)); !again {
		t.Fatal("did not re-notify after the cooldown expired")
	}
}

func TestAlertReArmsAfterDroppingBelowTheThreshold(t *testing.T) {
	start := time.Now()

	_, firing := shouldFireAlert(false, time.Time{}, 150, 100, start)

	// A counter reset drops the total below the line.
	dropped, firingAfterDrop := shouldFireAlert(firing, start, 10, 100, start.Add(time.Second))
	if dropped {
		t.Fatal("fired while below the threshold")
	}
	if firingAfterDrop {
		t.Fatal("still marked firing after dropping below the threshold")
	}

	// The next crossing is a new event, not the old one still ringing, so it
	// reports without waiting out the cooldown.
	again, _ := shouldFireAlert(firingAfterDrop, start, 150, 100, start.Add(2*time.Second))
	if !again {
		t.Fatal("a fresh crossing did not fire")
	}
}

// evaluateAlertThreshold is the part that owns the state; this is what proves
// the loop actually carries the decision between ticks.
func TestEvaluateAlertThresholdKeepsFiringStateBetweenTicks(t *testing.T) {
	server, _ := newTestServer(t)
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 100})
	start := time.Now()

	server.evaluateAlertThreshold(150, 100, start)

	server.alertMu.RLock()
	firing, lastAt := server.alertFiring, server.lastAlertAt
	server.alertMu.RUnlock()

	if !firing || lastAt.IsZero() {
		t.Fatalf("firing=%v lastAlertAt=%v, want the crossing recorded", firing, lastAt)
	}

	server.evaluateAlertThreshold(200, 100, start.Add(time.Minute))

	server.alertMu.RLock()
	unchanged := server.lastAlertAt.Equal(lastAt)
	server.alertMu.RUnlock()

	if !unchanged {
		t.Fatal("a tick inside the cooldown moved lastAlertAt, so it would re-notify")
	}
}
