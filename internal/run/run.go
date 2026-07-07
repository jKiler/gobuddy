// Package run executes external commands with the safety defaults gobuddy
// tools need: a bounded timeout and an explicit working directory, so a
// hung subprocess cannot stall an MCP session indefinitely.
package run

import (
	"context"
	"os/exec"
	"time"
)

// DefaultTimeout bounds every subprocess started through this package.
const DefaultTimeout = 30 * time.Second

// Command runs name with args in dir (the process working directory when
// dir is empty) and returns its combined stdout and stderr. The subprocess
// is killed once DefaultTimeout elapses or ctx is canceled, whichever
// comes first.
func Command(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
