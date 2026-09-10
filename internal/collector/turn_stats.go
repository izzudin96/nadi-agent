package collector

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// turnStatsAddr is coturn's CLI (telnet) port, where the `stats` command
// reports session counts and relayed bytes. Hardcoded to the default for v1;
// make configurable if a deployment uses a custom --cli-port.
const turnStatsAddr = "127.0.0.1:5766"

// readCoturnStats connects to coturn's CLI, sends `stats`, and parses the
// active-session count and relayed byte total. It returns as soon as both
// values are seen, rather than draining to the read deadline — coturn keeps
// the connection open at a prompt, so a plain ReadAll would burn the full 2s
// deadline every cycle. Any failure returns ok=false (best-effort).
func readCoturnStats(ctx context.Context, addr string) (sessions, bytes float64, ok bool) {
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, 0, false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte("stats\n")); err != nil {
		return 0, 0, false
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var buf strings.Builder
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
		if sessions, bytes, ok := parseCoturnStats(buf.String()); ok {
			return sessions, bytes, true
		}
	}
	// EOF or deadline: return whatever a final parse finds (possibly nothing).
	return parseCoturnStats(buf.String())
}

// parseCoturnStats extracts the session count and relayed-byte total from
// coturn's `stats` output. It is deliberately tolerant: it scans lines
// case-insensitively for the words "session" and "byte" and takes the first
// number on each. Verify against the deployed coturn version on the TURN server.
func parseCoturnStats(text string) (sessions, bytes float64, ok bool) {
	foundSessions, foundBytes := false, false
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if !foundSessions && strings.Contains(lower, "session") {
			if n, err := firstNumber(line); err == nil {
				sessions = n
				foundSessions = true
			}
		}
		if !foundBytes && strings.Contains(lower, "byte") {
			if n, err := firstNumber(line); err == nil {
				bytes = n
				foundBytes = true
			}
		}
	}
	return sessions, bytes, foundSessions && foundBytes
}

// firstNumber returns the first numeric token (optionally with a decimal
// point) on a line.
func firstNumber(line string) (float64, error) {
	i := strings.IndexFunc(line, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return 0, fmt.Errorf("no number in %q", line)
	}
	j := i
	for j < len(line) {
		c := line[j]
		if (c >= '0' && c <= '9') || c == '.' {
			j++
		} else {
			break
		}
	}
	return strconv.ParseFloat(line[i:j], 64)
}
