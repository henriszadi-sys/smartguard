//go:build !windows

package actions

import (
	"context"
	"os/exec"
)

// scriptCommand prépare l'exécution d'un script par le shell du système.
func scriptCommand(ctx context.Context, cmdline string) *exec.Cmd {
	return exec.CommandContext(ctx, "/bin/sh", "-c", cmdline)
}
