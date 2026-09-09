package sender

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/izzudin96/nadi-agent/internal/collector"
	"github.com/izzudin96/nadi-agent/internal/payload"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testMetrics() []collector.Metric {
	return []collector.Metric{
		{Name: "cpu.usage_percent", Value: 42.3, Unit: "%"},
		{Name: "battery.percent", Value: 87, Unit: "%"},
	}
}

func TestSendSuccess(t *testing.T) {
	var mu sync.Mutex
	var got payload.Heartbeat
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("bad JSON: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New("laptop-turn-01", "secret", server.URL, "dev", discardLogger(), server.Client())
	if err := s.Send(context.Background(), testMetrics()); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer secret")
	}
	if got.DeviceID != "laptop-turn-01" {
		t.Errorf("DeviceID = %q", got.DeviceID)
	}
	if len(got.Metrics) != 2 || got.Metrics[0].Name != "cpu.usage_percent" || got.Metrics[0].Value != 42.3 {
		t.Errorf("metrics mismatch: %+v", got.Metrics)
	}
	if got.Meta.Arch == "" || got.Meta.OS == "" || got.Meta.Hostname == "" || got.Meta.AgentVersion != "dev" {
		t.Errorf("meta incomplete: %+v", got.Meta)
	}
}

func TestSendNon2xxIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	s := New("d", "k", server.URL, "dev", discardLogger(), server.Client())
	err := s.Send(context.Background(), testMetrics())
	if err == nil {
		t.Fatal("Send() expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status, got: %v", err)
	}
}

func TestSendNetworkError(t *testing.T) {
	// URL that refuses connections.
	s := New("d", "k", "http://127.0.0.1:1", "dev", discardLogger(), http.DefaultClient)
	if err := s.Send(context.Background(), testMetrics()); err == nil {
		t.Fatal("Send() expected error for unreachable server, got nil")
	}
}

func TestSendDropsInvalidMetricNames(t *testing.T) {
	var mu sync.Mutex
	var got payload.Heartbeat

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New("d", "k", server.URL, "dev", discardLogger(), server.Client())
	metrics := []collector.Metric{
		{Name: "cpu.usage_percent", Value: 1},
		{Name: "BAD.NAME", Value: 1}, // invalid: uppercase
	}
	if err := s.Send(context.Background(), metrics); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(got.Metrics) != 1 || got.Metrics[0].Name != "cpu.usage_percent" {
		t.Errorf("expected invalid metric dropped, got %+v", got.Metrics)
	}
}
