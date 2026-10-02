// Package actions exécute les actions d'échéance : arrêt de services,
// exécution de scripts, et leur réactivation après renouvellement.
package actions

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"smartguard/internal/config"
)

// RunCmd exécute une commande avec délai maximal et renvoie un résumé pour le journal.
func RunCmd(timeout time.Duration, name string, args ...string) string {
	res, _ := run(timeout, name, args...)
	return res
}

func run(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return summarize(exec.CommandContext(ctx, name, args...), name+" "+strings.Join(args, " "))
}

func summarize(cmd *exec.Cmd, label string) (string, error) {
	out, err := cmd.CombinedOutput()
	res := strings.Join(strings.Fields(strings.ReplaceAll(string(out), "\n", " | ")), " ")
	if len(res) > 400 {
		res = res[:400] + "…"
	}
	if err != nil {
		return fmt.Sprintf("ÉCHEC %s : %v %s", label, err, res), err
	}
	return "OK " + label, nil
}

// StopService arrête le service et empêche son redémarrage automatique.
func StopService(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			RunCmd(60*time.Second, "sc.exe", "stop", name),
			RunCmd(30*time.Second, "sc.exe", "config", name, "start=", "disabled"),
		}
	}
	return []string{
		RunCmd(60*time.Second, "systemctl", "stop", name),
		RunCmd(30*time.Second, "systemctl", "disable", name),
	}
}

// RestoreService réactive et redémarre le service (après renouvellement).
func RestoreService(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			RunCmd(30*time.Second, "sc.exe", "config", name, "start=", "auto"),
			RunCmd(60*time.Second, "sc.exe", "start", name),
		}
	}
	return []string{
		RunCmd(30*time.Second, "systemctl", "enable", name),
		RunCmd(60*time.Second, "systemctl", "start", name),
	}
}

// Exécution des scripts : délai maximum par tentative, nombre de tentatives
// et pause entre deux tentatives.
const ScriptAttempts = 3

var (
	ScriptTimeout = 10 * time.Minute
	retryDelay    = 5 * time.Second
)

// RunScript exécute un script ou une commande (PowerShell pour les .ps1 sous Windows),
// en le retentant jusqu'à ScriptAttempts fois en cas d'échec.
func RunScript(cmdline string) string {
	var res string
	for i := 1; i <= ScriptAttempts; i++ {
		var err error
		res, err = runScriptOnce(cmdline)
		if err == nil {
			if i > 1 {
				return fmt.Sprintf("%s (tentative %d/%d)", res, i, ScriptAttempts)
			}
			return res
		}
		if i < ScriptAttempts {
			time.Sleep(retryDelay)
		}
	}
	return fmt.Sprintf("%s (après %d tentatives)", res, ScriptAttempts)
}

func runScriptOnce(cmdline string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ScriptTimeout)
	defer cancel()
	return summarize(scriptCommand(ctx, cmdline), cmdline)
}

// Logf : fonction de journalisation utilisée par les actions.
type Logf func(format string, args ...any)

// Enforce exécute les actions d'arrêt d'une échéance (une seule fois par date de fin).
func Enforce(c config.Config, d config.Deadline, logf Logf) {
	logf("ARRÊT PLANIFIÉ atteint pour « %s » — échéance « %s » (date de fin et d'arrêt : %s) — exécution des actions",
		c.SoftwareName, d.Name(), d.EndDate)
	for _, s := range d.Services {
		for _, line := range StopService(s) {
			logf("  service %s : %s", s, line)
		}
	}
	for _, sc := range d.Scripts {
		logf("  script : %s", RunScript(sc))
	}
	if len(d.BlockedURLs) > 0 {
		logf("  URL bloquées : %s", strings.Join(d.BlockedURLs, ", "))
	}
}

// Restore réactive les services des échéances indiquées après renouvellement
// (action explicite de l'administrateur). Un service commun à plusieurs
// échéances n'est relancé qu'une fois.
func Restore(ds []config.Deadline, logf Logf) {
	logf("RÉACTIVATION des services demandée par l'administrateur")
	seen := map[string]bool{}
	for _, d := range ds {
		for _, s := range d.Services {
			if seen[strings.ToLower(s)] {
				continue
			}
			seen[strings.ToLower(s)] = true
			for _, line := range RestoreService(s) {
				logf("  service %s (échéance « %s ») : %s", s, d.Name(), line)
			}
		}
	}
}
