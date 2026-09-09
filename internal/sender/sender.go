package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/izzudin96/nadi-agent/internal/collector"
	"github.com/izzudin96/nadi-agent/internal/payload"
)

// Sender serializes collected metrics into a heartbeat and POSTs it to the
// server. It is built once at startup and reused every cycle.
type Sender struct {
	deviceID  string
	apiKey    string
	serverURL string
	version   string
	logger    *slog.Logger
	client    *http.Client
}

// New builds a Sender. client is injected so tests can point it at a mock
// server; pass http.DefaultClient in production.
func New(deviceID, apiKey, serverURL, version string, logger *slog.Logger, client *http.Client) *Sender {
	if client == nil {
		client = http.DefaultClient
	}
	return &Sender{
		deviceID:  deviceID,
		apiKey:    apiKey,
		serverURL: serverURL,
		version:   version,
		logger:    logger,
		client:    client,
	}
}

// Build serializes the collected metrics into a heartbeat JSON payload.
func (s *Sender) Build(metrics []collector.Metric) ([]byte, error) {
	heartbeat, err := s.buildHeartbeat(metrics)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(heartbeat)
	if err != nil {
		return nil, fmt.Errorf("marshaling heartbeat: %w", err)
	}
	return data, nil
}

// SendRaw POSTs a pre-built payload. It returns an error for network failures
// and non-2xx responses so the caller can decide what to do (buffer + retry).
func (s *Sender) SendRaw(ctx context.Context, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("posting heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

// Send builds and POSTs a heartbeat in one step. Convenience wrapper around
// Build + SendRaw; the agent uses the two-step form so it can buffer failures.
func (s *Sender) Send(ctx context.Context, metrics []collector.Metric) error {
	data, err := s.Build(metrics)
	if err != nil {
		return err
	}
	return s.SendRaw(ctx, data)
}

func (s *Sender) buildHeartbeat(metrics []collector.Metric) (*payload.Heartbeat, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("getting hostname: %w", err)
	}

	pmetrics := make([]payload.Metric, 0, len(metrics))
	for _, m := range metrics {
		if !payload.ValidMetricName(m.Name) {
			s.logger.Debug("dropping metric with invalid name", "name", m.Name)
			continue
		}
		pmetrics = append(pmetrics, payload.Metric{Name: m.Name, Value: m.Value, Unit: m.Unit})
		if len(pmetrics) >= payload.MaxMetrics {
			s.logger.Debug("payload at metric cap, truncating", "cap", payload.MaxMetrics)
			break
		}
	}

	return &payload.Heartbeat{
		DeviceID:  s.deviceID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metrics:   pmetrics,
		Meta: payload.Meta{
			Hostname:     hostname,
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			AgentVersion: s.version,
		},
	}, nil
}
