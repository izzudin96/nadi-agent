package collector

import (
	"context"
	"testing"
)

func TestCoturnProcessNames(t *testing.T) {
	if !coturnProcessNames["coturn"] || !coturnProcessNames["turnserver"] {
		t.Fatalf("coturn/turnserver should be recognized")
	}
	for _, name := range []string{"nginx", "Coturn", "", "coturn2"} {
		if coturnProcessNames[name] {
			t.Errorf("name %q should not be recognized as coturn", name)
		}
	}
}

// TestTurnProcessAbsent verifies that when coturn is not running (as in CI and
// on a dev Mac), the collector emits turn.process_up=0 and nothing else.
func TestTurnProcessAbsent(t *testing.T) {
	reg, err := NewRegistry(map[string]bool{"turn_process": true}, discardLogger())
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	metrics := reg.Collect(context.Background())

	if len(metrics) != 1 {
		t.Fatalf("got %d metrics, want 1: %v", len(metrics), metrics)
	}
	m := metrics[0]
	if m.Name != "turn.process_up" || m.Value != 0 {
		t.Fatalf("metric = %+v, want turn.process_up=0", m)
	}
}
