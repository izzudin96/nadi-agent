package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validYAML = `
device_id: "laptop-turn-01"
server_url: "https://monitor.example.com/api/heartbeat"
api_key: "secret-token"
interval_seconds: 15
jitter_seconds: 3
buffer_path: "./agent-buffer.db"
buffer_max_size_mb: 50
log_level: "info"
collectors:
  cpu: true
  memory: true
  battery: true
`

func TestLoadValid(t *testing.T) {
	path := writeConfig(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DeviceID != "laptop-turn-01" {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, "laptop-turn-01")
	}
	if cfg.IntervalSeconds != 15 {
		t.Errorf("IntervalSeconds = %d, want 15", cfg.IntervalSeconds)
	}
	if cfg.JitterSeconds != 3 {
		t.Errorf("JitterSeconds = %d, want 3", cfg.JitterSeconds)
	}
	if !cfg.Collectors["cpu"] {
		t.Error("collectors.cpu should be enabled")
	}
}

func TestLoadMissingRequired(t *testing.T) {
	path := writeConfig(t, `
server_url: "https://monitor.example.com/api/heartbeat"
api_key: "secret"
interval_seconds: 15
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "device_id") {
		t.Errorf("error should mention device_id, got: %v", err)
	}
}

func TestLoadInvalidServerURL(t *testing.T) {
	path := writeConfig(t, `
device_id: "x"
server_url: "not-a-url"
api_key: "secret"
interval_seconds: 15
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "server_url") {
		t.Fatalf("expected server_url error, got: %v", err)
	}
}

func TestLoadInvalidInterval(t *testing.T) {
	path := writeConfig(t, `
device_id: "x"
server_url: "https://monitor.example.com/api/heartbeat"
api_key: "secret"
interval_seconds: 0
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "interval_seconds") {
		t.Fatalf("expected interval_seconds error, got: %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("Load() expected error for missing file, got nil")
	}
}
