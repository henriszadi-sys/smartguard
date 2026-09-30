//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"smartguard/internal/actions"
)

func IsAdmin() bool { return os.Geteuid() == 0 }

func RelaunchElevated(args []string) error {
	return errors.New("relancez avec : sudo " + os.Args[0])
}

func OpenBrowser(url string) {
	for _, c := range []string{"xdg-open", "open"} {
		if p, err := exec.LookPath(c); err == nil {
			cmd := exec.Command(p, url)
			// sous sudo, ouvrir le navigateur de l'utilisateur d'origine
			if u := os.Getenv("SUDO_USER"); u != "" {
				if su, err := exec.LookPath("sudo"); err == nil {
					cmd = exec.Command(su, "-u", u, p, url)
				}
			}
			_ = cmd.Start()
			return
		}
	}
}

func HasDesktop() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func RegistryDir() string { return "/etc/smartguard" }

// LegacyRegistryDir : registre des versions publiées sous l'ancien nom du produit.
func LegacyRegistryDir() string { return "/etc/licguard" }

func DefaultInstallDir(name string) string { return filepath.Join("/opt", strings.ToLower(name)) }

func ExeFileName(name string) string { return strings.ToLower(name) }

type SysService struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

func ListSystemServices() []SysService {
	out, err := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--plain", "--no-pager").Output()
	if err != nil {
		return nil
	}
	var list []SysService
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		name := strings.TrimSuffix(f[0], ".service")
		desc := strings.Join(f[4:], " ")
		st := "arrêté"
		if f[2] == "active" {
			st = "démarré"
		}
		list = append(list, SysService{Name: name, DisplayName: desc, Status: st})
	}
	return list
}

func OpenFirewall(name string, port int) string {
	if _, err := exec.LookPath("ufw"); err == nil {
		return actions.RunCmd(30*time.Second, "ufw", "allow", fmt.Sprintf("%d/tcp", port))
	}
	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		r := actions.RunCmd(30*time.Second, "firewall-cmd", "--permanent", fmt.Sprintf("--add-port=%d/tcp", port))
		_ = exec.Command("firewall-cmd", "--reload").Run()
		return r
	}
	return "aucun pare-feu géré détecté (ufw / firewalld) — rien à faire"
}

func CloseFirewall(name string) {}

func CreateShortcut(name, url string) error { return nil }
func RemoveShortcut(name string)            {}
func PauseConsole()                         {}

func ServiceDiagnostics(name string) string {
	out, _ := exec.Command("systemctl", "status", name, "--no-pager", "-n", "5").CombinedOutput()
	return "• État du service :\n" + strings.TrimSpace(string(out)) + "\n"
}
