//go:build !windows

package license

import (
	"errors"
	"os"
	"strings"
)

// MachineID renvoie l'identifiant du poste (machine-id systemd / D-Bus, haché).
func MachineID() (string, error) {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(p); err == nil && strings.TrimSpace(string(b)) != "" {
			return hashMachine(string(b)), nil
		}
	}
	return "", errors.New("aucun identifiant machine trouvé (/etc/machine-id)")
}
