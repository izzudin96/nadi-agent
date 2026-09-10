package collector

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

func init() {
	Register("network", NewNetworkCollector)
}

// publicIPEndpoint returns the machine's public IP as plain text. Used by the
// public_ip_changed metric, which matters for TURN servers whose IP may rotate.
const publicIPEndpoint = "https://api.ipify.org"

type networkCollector struct {
	logger     *slog.Logger
	httpClient *http.Client

	// Previous counters, kept to compute per-second rates. This is why the
	// collector is stateful: a rate needs a delta since the last tick.
	prevBytesSent uint64
	prevBytesRecv uint64
	prevTime      time.Time

	lastPublicIP string
}

func NewNetworkCollector(logger *slog.Logger) Collector {
	return &networkCollector{
		logger:     logger,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *networkCollector) Name() string { return "network" }

func (c *networkCollector) Collect(ctx context.Context) ([]Metric, error) {
	counters, err := net.IOCountersWithContext(ctx, false)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	io := counters[0] // pernic=false: single aggregate across all interfaces

	metrics := []Metric{
		{Name: "network.bytes_sent_per_sec", Value: c.rate(now, c.prevBytesSent, io.BytesSent), Unit: "bytes/s"},
		{Name: "network.bytes_recv_per_sec", Value: c.rate(now, c.prevBytesRecv, io.BytesRecv), Unit: "bytes/s"},
	}
	c.prevBytesSent = io.BytesSent
	c.prevBytesRecv = io.BytesRecv
	c.prevTime = now

	// External IP check is best-effort: no public internet or a slow response
	// simply omits the metric rather than failing the collector.
	if changed, ok := c.publicIPChanged(ctx); ok {
		metrics = append(metrics, Metric{Name: "network.public_ip_changed", Value: changed, Unit: "bool"})
	}

	return metrics, nil
}

// rate computes bytes-per-second between the previous and current counters.
func (c *networkCollector) rate(now time.Time, prev, cur uint64) float64 {
	if c.prevTime.IsZero() {
		return 0 // first cycle: no baseline to measure against yet
	}
	if cur < prev {
		return 0 // counter reset (interface restarted); ignore this cycle
	}
	dt := now.Sub(c.prevTime).Seconds()
	if dt <= 0 {
		return 0
	}
	return float64(cur-prev) / dt
}

func (c *networkCollector) publicIPChanged(ctx context.Context) (float64, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, publicIPEndpoint, nil)
	if err != nil {
		return 0, false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("public IP check failed", "err", err)
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("public IP check failed", "status", resp.StatusCode)
		return 0, false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false
	}
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return 0, false
	}

	changed := 0.0
	if c.lastPublicIP != "" && ip != c.lastPublicIP {
		changed = 1
	}
	c.lastPublicIP = ip
	return changed, true
}
