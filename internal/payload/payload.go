package payload

import "regexp"

// Metric is one collected value in the wire format. Mirrors the JSON schema in
// spec §6: name/value/unit, no labels.
type Metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

// Meta is sent with every heartbeat (cheap, small) rather than a separate
// registration call.
type Meta struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	AgentVersion string `json:"agent_version"`
}

// Heartbeat is the full payload POSTed to server_url.
type Heartbeat struct {
	DeviceID  string   `json:"device_id"`
	Timestamp string   `json:"timestamp"`
	Metrics   []Metric `json:"metrics"`
	Meta      Meta     `json:"meta"`
}

const (
	// MaxMetrics caps the number of metrics per payload (spec §6).
	MaxMetrics = 100
	// MaxNameLen caps a metric name length (spec §6).
	MaxNameLen = 64
)

var metricNameRe = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)+$`)

// ValidMetricName enforces the naming contract shared by agent and server
// (plan §4.3): lowercase, dot-separated segments, no leading/trailing dots.
func ValidMetricName(name string) bool {
	return len(name) <= MaxNameLen && metricNameRe.MatchString(name)
}
