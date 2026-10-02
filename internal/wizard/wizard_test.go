package wizard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// Sauvegarde automatique avant mise à jour : copie horodatée, 10 au maximum.
func TestBackupConfigKeepsLatest(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if dst, err := backupConfig(cfg, time.Now()); err != nil || dst != "" {
		t.Fatalf("aucune configuration : %q %v", dst, err)
	}
	if err := os.WriteFile(cfg, []byte(`{"module_name":"Kelio"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 10, 2, 8, 0, 0, 0, time.Local)
	var last string
	for i := 0; i < MaxBackups+3; i++ {
		dst, err := backupConfig(cfg, t0.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		last = dst
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(cfg), "sauvegardes", "config-*.json"))
	if len(files) != MaxBackups {
		t.Fatalf("%d sauvegardes conservées, attendu %d", len(files), MaxBackups)
	}
	if b, _ := os.ReadFile(last); string(b) != `{"module_name":"Kelio"}` {
		t.Fatalf("contenu de la sauvegarde : %s", b)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(cfg), "sauvegardes", "config-20261002-080000.json")); err == nil {
		t.Fatal("la plus ancienne sauvegarde n'a pas été supprimée")
	}
}

// Système non pris en charge : installation refusée avant toute copie de fichier.
func TestInstallRefusedOnUnsupportedSystem(t *testing.T) {
	old := systemCheck
	systemCheck = func() (bool, string) {
		return false, "Windows 6.3 (build 9600) : Windows 10 ou Windows Server 2016 minimum requis"
	}
	defer func() { systemCheck = old }()
	dir := filepath.Join(t.TempDir(), "mod")
	_, err := doInstall(installReq{ModuleName: "TestSystemeZ9", SoftwareName: "Kelio", Port: 47911, Mode: "script",
		EndDate: "2027-01-01", Password: "motdepasse1", InstallDir: dir})
	if err == nil || !strings.Contains(err.Error(), "Windows 10") {
		t.Fatalf("installation sur système non pris en charge : %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("des fichiers ont été copiés malgré le refus")
	}
}
