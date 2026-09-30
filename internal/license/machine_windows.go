//go:build windows

package license

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// MachineID renvoie l'identifiant du poste (MachineGuid Windows, haché).
func MachineID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer k.Close()
	v, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	if v == "" {
		return "", errors.New("MachineGuid vide")
	}
	return hashMachine(v), nil
}
