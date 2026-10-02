package wizard

import (
	"crypto/tls"
	"crypto/x509"
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

// HTTPS : certificat auto-signé créé une fois, réutilisé ensuite, utilisable par le serveur.
func TestSelfSignedCertificate(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.Local)
	c1, k1, err := ensureSelfSigned(dir, []string{"srv-kelio", "192.168.1.20", "localhost"}, now)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.LoadX509KeyPair(c1, k1)
	if err != nil {
		t.Fatalf("certificat inutilisable : %v", err)
	}
	leaf, _ := x509.ParseCertificate(pair.Certificate[0])
	if err := leaf.VerifyHostname("srv-kelio"); err != nil {
		t.Fatalf("nom du serveur absent du certificat : %v", err)
	}
	if err := leaf.VerifyHostname("192.168.1.20"); err != nil {
		t.Fatalf("adresse du serveur absente du certificat : %v", err)
	}
	b1, _ := os.ReadFile(c1)
	c2, _, err := ensureSelfSigned(dir, []string{"srv-kelio"}, now.Add(24*time.Hour))
	if b2, _ := os.ReadFile(c2); err != nil || string(b1) != string(b2) {
		t.Fatal("certificat valide régénéré au lieu d'être réutilisé")
	}
	if !isGeneratedTLS(dir, c1) || isGeneratedTLS(dir, filepath.Join(dir, "client.pem")) {
		t.Fatal("reconnaissance du certificat généré")
	}
	if Scheme(c1, k1) != "https" || Scheme("", "") != "http" {
		t.Fatal("schéma d'adresse")
	}
}
