// Command mockserver is a minimal stand-in for nadi-server used to develop
// and test the agent. It receives heartbeats, records them, and exposes a
// debug endpoint. See PLAN §7.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/izzudin96/nadi-agent/internal/payload"
)

type recorder struct {
	mu       sync.Mutex
	received []payload.Heartbeat
	fail     bool
	delay    time.Duration
}

func main() {
	addr := flag.String("addr", ":9100", "listen address")
	flag.Parse()

	r := &recorder{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/heartbeat", r.handleHeartbeat)
	mux.HandleFunc("/debug/received", r.handleReceived)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("mockserver listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (r *recorder) handleHeartbeat(w http.ResponseWriter, req *http.Request) {
	// Optional fault injection for buffer/backoff testing (Phase 4):
	// /api/heartbeat?fail=1 or /api/heartbeat?delay=3s
	q := req.URL.Query()
	if q.Get("fail") == "1" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "injected failure")
		return
	}
	if d := q.Get("delay"); d != "" {
		wait, err := time.ParseDuration(d)
		if err == nil {
			time.Sleep(wait)
		}
	}

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if auth := req.Header.Get("Authorization"); !strings.HasPrefix(auth, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "missing bearer token")
		return
	}

	var hb payload.Heartbeat
	if err := json.NewDecoder(req.Body).Decode(&hb); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "bad payload: %v", err)
		return
	}

	r.mu.Lock()
	r.received = append(r.received, hb)
	n := len(r.received)
	r.mu.Unlock()

	log.Printf("heartbeat %d from %s: %d metrics", n, hb.DeviceID, len(hb.Metrics))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok (%d)", n)
}

func (r *recorder) handleReceived(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(r.received); err != nil {
		log.Printf("encoding received: %v", err)
	}
}
