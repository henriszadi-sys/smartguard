package wizard

import (
	"errors"
	"path/filepath"
	"testing"

	"smartguard/internal/license"
)

func testManager(t *testing.T) *license.Manager {
	t.Helper()
	return &license.Manager{Path: filepath.Join(t.TempDir(), "license.json"), Machine: func() (string, error) { return "srv-1", nil }}
}

// Nouvelle installation : refusée sans licence valide, acceptée avec une clé valide.
func TestNewInstallRequiresLicense(t *testing.T) {
	lm := testManager(t)
	if _, err := checkLicense(lm, "", false); !errors.Is(err, license.ErrLicenseRequired) {
		t.Fatalf("installation sans licence : %v", err)
	}
	if _, err := checkLicense(lm, "SGRD-AAAA-BBBB-CCCC-0000", false); err == nil {
		t.Fatal("installation avec une clé invalide acceptée")
	}
	if _, err := checkLicense(lm, license.MakeKey("ABCD", "EFGH", "JKLM"), false); err != nil {
		t.Fatalf("installation avec clé valide : %v", err)
	}
	if _, err := checkLicense(lm, "", false); err != nil {
		t.Fatalf("second module sur un serveur licencié : %v", err)
	}
}

// Mise à jour d'un module existant sans licence : acceptée par défaut, mais signalée.
func TestUpdateWithoutLicense(t *testing.T) {
	lm := testManager(t)
	detail, err := checkLicense(lm, "", true)
	if RequireLicenseOnUpdate {
		if err == nil {
			t.Fatal("mise à jour sans licence acceptée alors qu'elle est exigée")
		}
		return
	}
	if err != nil || detail == "" {
		t.Fatalf("mise à jour sans licence : %q %v", detail, err)
	}
	// Une clé invalide saisie lors d'une mise à jour reste refusée.
	if _, err := checkLicense(lm, "SGRD-AAAA-BBBB-CCCC-0000", true); err == nil {
		t.Fatal("clé invalide acceptée lors d'une mise à jour")
	}
}
