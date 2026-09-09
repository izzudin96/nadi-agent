package collector

import (
	"context"
	"os/exec"
)

// commandExists reports whether a binary is on PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runCommand runs a command with the cycle's context (which carries the
// collector deadline) and returns its stdout. A non-zero exit or a missing
// binary surfaces as a non-nil error.
func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
