package collector

import (
	"context"
	"fmt"
	"net"
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

func TestParseCoturnStats(t *testing.T) {
	text := "Server stats:\nTotal sessions: 34\nRelayed bytes: 123456789\nUptime: 999\n"
	sessions, bytes, ok := parseCoturnStats(text)
	if !ok {
		t.Fatal("parseCoturnStats() not ok")
	}
	if sessions != 34 {
		t.Errorf("sessions = %v, want 34", sessions)
	}
	if bytes != 123456789 {
		t.Errorf("bytes = %v, want 123456789", bytes)
	}
}

func TestParseCoturnStatsPartial(t *testing.T) {
	// Only sessions present -> not ok (bytes missing).
	_, _, ok := parseCoturnStats("Total sessions: 5\n")
	if ok {
		t.Fatal("expected not ok when bytes missing")
	}
	// Nothing relevant.
	if _, _, ok := parseCoturnStats("nothing useful here"); ok {
		t.Fatal("expected not ok for empty stats")
	}
}

func TestReadCoturnStats(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 64)
		_, _ = conn.Read(buf) // read the "stats" command
		fmt.Fprintf(conn, "Total sessions: 42\nTotal relayed bytes: 987654\n")
	}()

	sessions, bytes, ok := readCoturnStats(context.Background(), ln.Addr().String())
	if !ok {
		t.Fatal("readCoturnStats() not ok")
	}
	if sessions != 42 || bytes != 987654 {
		t.Fatalf("got sessions=%v bytes=%v, want 42/987654", sessions, bytes)
	}
}

func TestReadCoturnStatsUnreachable(t *testing.T) {
	if _, _, ok := readCoturnStats(context.Background(), "127.0.0.1:1"); ok {
		t.Fatal("expected not ok for unreachable CLI")
	}
}
