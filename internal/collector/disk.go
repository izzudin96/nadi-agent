package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

func init() {
	Register("disk", NewDiskCollector)
}

// virtualFSTypes are pseudo filesystems that have no real storage behind them.
var virtualFSTypes = map[string]bool{
	"devfs": true, "devtmpfs": true, "proc": true, "sysfs": true,
	"tmpfs": true, "devpts": true, "cgroup": true, "cgroup2": true,
	"mqueue": true, "shm": true, "autofs": true, "overlay": true,
	"ramfs": true, "securityfs": true, "debugfs": true, "tracefs": true,
	"fusectl": true, "configfs": true, "hugetlbfs": true, "pstore": true,
	"binfmt_misc": true,
}

// virtualMountPrefixes are mount points that are never worth reporting usage for.
var virtualMountPrefixes = []string{"/dev", "/proc", "/sys", "/System/Volumes"}

type diskCollector struct{}

func NewDiskCollector(_ *slog.Logger) Collector { return &diskCollector{} }

func (c *diskCollector) Name() string { return "disk" }

func (c *diskCollector) Collect(ctx context.Context) ([]Metric, error) {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	for _, p := range partitions {
		if !shouldReport(p.Mountpoint, p.Fstype) {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue // volume unavailable right now; skip it, not the cycle
		}
		metrics = append(metrics, Metric{
			Name:  "disk.usage_percent." + sanitize(p.Mountpoint),
			Value: usage.UsedPercent,
			Unit:  "%",
		})
	}
	return metrics, nil
}

func shouldReport(mountpoint, fstype string) bool {
	if virtualFSTypes[fstype] {
		return false
	}
	for _, prefix := range virtualMountPrefixes {
		if mountpoint == prefix || strings.HasPrefix(mountpoint, prefix+"/") {
			return false
		}
	}
	return true
}

// sanitize turns a mount point into a metric-name-safe tag, e.g.
// "/" -> "root", "/home/user" -> "home_user".
func sanitize(mountpoint string) string {
	parts := strings.FieldsFunc(strings.ToLower(mountpoint), func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return "root"
	}
	for i, part := range parts {
		parts[i] = sanitizePart(part)
	}
	return strings.Join(parts, "_")
}

func sanitizePart(part string) string {
	var b strings.Builder
	for _, r := range part {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
