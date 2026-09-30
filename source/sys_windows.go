//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func isAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// relaunchElevated relance l'exécutable avec l'invite UAC « Exécuter en tant qu'administrateur ».
func relaunchElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = syscall.EscapeArg(a)
	}
	params, _ := syscall.UTF16PtrFromString(strings.Join(quoted, " "))
	cwd, _ := syscall.UTF16PtrFromString(filepath.Dir(exe))
	return windows.ShellExecute(0, verb, file, params, cwd, windows.SW_NORMAL)
}

func openBrowser(url string) {
	// explorer.exe délègue au shell de l'utilisateur : le navigateur ne s'ouvre pas en mode administrateur.
	_ = exec.Command("explorer.exe", url).Start()
}

func hasDesktop() bool { return true }

func registryDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "LicGuard")
}

func defaultInstallDir(name string) string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	return filepath.Join(pf, name)
}

func exeFileName(name string) string { return name + ".exe" }

type sysService struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

func listSystemServices() []sysService {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		"[Console]::OutputEncoding=[Text.Encoding]::UTF8; Get-Service | Sort-Object DisplayName | ForEach-Object { [pscustomobject]@{name=$_.Name; display_name=$_.DisplayName; status=$_.Status.ToString()} } | ConvertTo-Json -Compress").Output()
	if err != nil {
		return nil
	}
	var list []sysService
	if json.Unmarshal(out, &list) != nil {
		var one sysService
		if json.Unmarshal(out, &one) == nil {
			list = []sysService{one}
		}
	}
	for i := range list {
		switch strings.ToLower(list[i].Status) {
		case "running":
			list[i].Status = "démarré"
		case "stopped":
			list[i].Status = "arrêté"
		}
	}
	return list
}

func openFirewall(name string, port int) string {
	_ = exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name).Run()
	return runCmd(30*time.Second, "netsh", "advfirewall", "firewall", "add", "rule", "name="+name,
		"dir=in", "action=allow", "protocol=TCP", fmt.Sprintf("localport=%d", port))
}

func closeFirewall(name string) {
	_ = exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name).Run()
}

func shortcutPath(name string) string {
	pub := os.Getenv("PUBLIC")
	if pub == "" {
		pub = `C:\Users\Public`
	}
	return filepath.Join(pub, "Desktop", name+" - administration.url")
}

// createShortcut crée un raccourci sur le bureau commun vers la page d'administration.
func createShortcut(name, url string) error {
	return os.WriteFile(shortcutPath(name), []byte("[InternetShortcut]\r\nURL="+url+"\r\n"), 0644)
}

func removeShortcut(name string) { _ = os.Remove(shortcutPath(name)) }

func pauseConsole() {
	fmt.Print("\nAppuyez sur Entrée pour fermer…")
	_, _ = fmt.Scanln()
}

// serviceDiagnostics : état du service et derniers événements du gestionnaire de services Windows.
func serviceDiagnostics(name string) string {
	out, _ := exec.Command("sc.exe", "query", name).CombinedOutput()
	res := "• État du service (sc query " + name + ") :\n" + strings.TrimSpace(string(out)) + "\n"
	ps := "[Console]::OutputEncoding=[Text.Encoding]::UTF8; Get-WinEvent -FilterHashtable @{LogName='System';ProviderName='Service Control Manager';StartTime=(Get-Date).AddMinutes(-15)} -ErrorAction SilentlyContinue | Where-Object { $_.Message -like '*" + name + "*' } | Select-Object -First 3 | ForEach-Object { $_.TimeCreated.ToString('HH:mm:ss') + ' ' + $_.Message }"
	ev, _ := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", ps).CombinedOutput()
	if e := strings.TrimSpace(string(ev)); e != "" {
		res += "• Journal d'événements Windows :\n" + e + "\n"
	}
	return res
}
