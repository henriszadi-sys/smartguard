package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// runCmd exécute une commande avec délai maximal et renvoie un résumé pour le journal.
func runCmd(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	res := strings.Join(strings.Fields(strings.ReplaceAll(string(out), "\n", " | ")), " ")
	if len(res) > 400 {
		res = res[:400] + "…"
	}
	if err != nil {
		return fmt.Sprintf("ÉCHEC %s %s : %v %s", name, strings.Join(args, " "), err, res)
	}
	return fmt.Sprintf("OK %s %s", name, strings.Join(args, " "))
}

// stopService arrête le service et empêche son redémarrage automatique.
func stopService(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			runCmd(60*time.Second, "sc.exe", "stop", name),
			runCmd(30*time.Second, "sc.exe", "config", name, "start=", "disabled"),
		}
	}
	return []string{
		runCmd(60*time.Second, "systemctl", "stop", name),
		runCmd(30*time.Second, "systemctl", "disable", name),
	}
}

// restoreService réactive et redémarre le service (après renouvellement).
func restoreService(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			runCmd(30*time.Second, "sc.exe", "config", name, "start=", "auto"),
			runCmd(60*time.Second, "sc.exe", "start", name),
		}
	}
	return []string{
		runCmd(30*time.Second, "systemctl", "enable", name),
		runCmd(60*time.Second, "systemctl", "start", name),
	}
}

func runScript(cmdline string) string {
	if runtime.GOOS == "windows" {
		l := strings.ToLower(cmdline)
		if strings.HasSuffix(strings.Trim(l, `" `), ".ps1") {
			return runCmd(10*time.Minute, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", strings.Trim(cmdline, `"`))
		}
		return runCmd(10*time.Minute, "cmd.exe", "/C", cmdline)
	}
	return runCmd(10*time.Minute, "/bin/sh", "-c", cmdline)
}

// enforceExpiry : actions exécutées une seule fois à la date d'expiration.
func (a *App) enforceExpiry(c Config) {
	a.logf("EXPIRATION atteinte pour « %s » (fin : %s) — exécution des actions", c.SoftwareName, c.EndDate)
	for _, s := range c.Services {
		for _, line := range stopService(s) {
			a.logf("  service %s : %s", s, line)
		}
	}
	for _, sc := range c.Scripts {
		a.logf("  script : %s", runScript(sc))
	}
	if len(c.BlockedURLs) > 0 {
		a.logf("  URL bloquées : %s", strings.Join(c.BlockedURLs, ", "))
	}
}

func (a *App) restoreAfterRenewal(c Config) {
	a.logf("RÉACTIVATION des services demandée par l'administrateur")
	for _, s := range c.Services {
		for _, line := range restoreService(s) {
			a.logf("  service %s : %s", s, line)
		}
	}
}

// watch vérifie l'échéance toutes les 30 secondes.
func (a *App) watch(stop <-chan struct{}) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	a.tick()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			a.tick()
		}
	}
}

func (a *App) tick() {
	st := a.store.State()
	now := time.Now()
	if now.After(st.LastSeen) {
		a.store.UpdateState(func(s *State) { s.LastSeen = now })
		st = a.store.State()
	}
	c := a.store.Config()
	if !c.Enabled {
		return
	}
	s := computeStatus(c, st)
	if s.Expired && (st.ActionsDoneAt.IsZero() || st.ActionsFor != c.EndDate) {
		a.store.UpdateState(func(x *State) { x.ActionsDoneAt = time.Now(); x.ActionsFor = c.EndDate })
		a.enforceExpiry(c)
	}
}
