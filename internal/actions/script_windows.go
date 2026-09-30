//go:build windows

package actions

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
)

// scriptCommand prépare l'exécution d'un script. Les .ps1 passent par PowerShell ;
// le reste est transmis tel quel à cmd.exe, car l'échappement standard de Go
// (\") n'est pas compris par cmd et casserait les chemins entre guillemets.
func scriptCommand(ctx context.Context, cmdline string) *exec.Cmd {
	if strings.HasSuffix(strings.ToLower(strings.Trim(cmdline, `" `)), ".ps1") {
		return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", strings.Trim(cmdline, `"`))
	}
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /S /C "` + cmdline + `"`}
	return cmd
}
