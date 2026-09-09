package payload

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidMetricName(t *testing.T) {
	valid := []string{
		"cpu.usage_percent",
		"disk.usage_percent.root",
		"process.top_cpu.google_chrome.cpu_percent",
		"battery.cycle_count",
	}
	for _, name := range valid {
		if !ValidMetricName(name) {
			t.Errorf("ValidMetricName(%q) = false, want true", name)
		}
	}

	invalid := []string{
		"",           // empty
		"cpu",        // no dot
		"cpu.",       // trailing dot
		".cpu",       // leading dot
		"CPU.usage",  // uppercase
		"cpu.usage%", // invalid char
		"cpu..usage", // empty segment
		"cpu.usage.percent." + strings.Repeat("x", 60), // > 64 chars total
	}
	for _, name := range invalid {
		if ValidMetricName(name) {
			t.Errorf("ValidMetricName(%q) = true, want false", name)
		}
	}
}

func TestHeartbeatJSONRoundTrip(t *testing.T) {
	h := Heartbeat{
		DeviceID:  "laptop-turn-01",
		Timestamp: "2026-08-24T10:30:00Z",
		Metrics: []Metric{
			{Name: "cpu.usage_percent", Value: 42.3, Unit: "%"},
			{Name: "battery.charging", Value: 1, Unit: "bool"},
		},
		Meta: Meta{Hostname: "turn-01", OS: "darwin", Arch: "arm64", AgentVersion: "0.1.0"},
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got Heartbeat
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.DeviceID != h.DeviceID || got.Meta.OS != "darwin" {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if len(got.Metrics) != 2 || got.Metrics[0].Name != "cpu.usage_percent" {
		t.Errorf("metrics mismatch: %+v", got.Metrics)
	}
}

func TestHeartbeatJSONFieldNames(t *testing.T) {
	data, err := json.Marshal(Heartbeat{
		DeviceID:  "d1",
		Timestamp: "t",
		Metrics:   []Metric{{Name: "cpu.usage_percent", Value: 1}},
		Meta:      Meta{Hostname: "h"},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{`"device_id"`, `"timestamp"`, `"metrics"`, `"hostname"`, `"agent_version"`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %s: %s", want, s)
		}
	}
}
