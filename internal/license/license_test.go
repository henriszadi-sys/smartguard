package license

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local)

func newManager(t *testing.T, machine *string) *Manager {
	t.Helper()
	return &Manager{
		Path:    filepath.Join(t.TempDir(), "config.license.json"),
		Now:     func() time.Time { return t0 },
		Machine: func() (string, error) { return *machine, nil },
	}
}

func TestKeyFormatAndChecksum(t *testing.T) {
	k := MakeKey("ABCD", "EFGH", "JKLM")
	if !CheckKey(k) {
		t.Fatalf("clé générée refusée : %s", k)
	}
	if got := NormalizeKey("  " + strings.ToLower(k) + " "); got != k {
		t.Fatalf("normalisation : %q", got)
	}
	bad := []string{"", "SGRD", k[:len(k)-1], strings.Replace(k, "ABCD", "ABCE", 1), "XXXX" + k[4:], "SGRD-ABCD-EFGH-JKLM-0000", "SGRD-abcd-EFGH-JKLM-" + k[len(k)-4:]}
	for _, b := range bad {
		if CheckKey(b) {
			t.Errorf("clé invalide acceptée : %q", b)
		}
	}
}

func TestMaskHidesKey(t *testing.T) {
	k := MakeKey("ABCD", "EFGH", "JKLM")
	m := Mask(k)
	if strings.Contains(m, "ABCD") || strings.Contains(m, "EFGH") || strings.Contains(m, "JKLM") || !strings.HasSuffix(m, k[len(k)-4:]) {
		t.Fatalf("masquage incorrect : %s", m)
	}
}

func TestActivateBindsToMachine(t *testing.T) {
	mid := "poste-a"
	m := newManager(t, &mid)
	if s := m.Status(); s.State != StateNone {
		t.Fatalf("état initial : %+v", s)
	}
	key := MakeKey("ABCD", "EFGH", "JKLM")
	s, err := m.Activate(strings.ToLower(key))
	if err != nil || s.State != StateActive || s.MachineID != "poste-a" {
		t.Fatalf("activation : %+v, %v", s, err)
	}
	if strings.Contains(s.MaskedKey, "ABCD") {
		t.Fatalf("clé non masquée : %s", s.MaskedKey)
	}
	if s.ActivatedAt != "30/09/2026 10:00" {
		t.Fatalf("date d'activation (horloge injectée) : %q", s.ActivatedAt)
	}
	// Réactiver la même licence sur le même poste est sans effet.
	if _, err := m.Activate(key); err != nil {
		t.Fatalf("réactivation idempotente refusée : %v", err)
	}
}

func TestInvalidKeyRejected(t *testing.T) {
	mid := "poste-a"
	m := newManager(t, &mid)
	if _, err := m.Activate("SGRD-AAAA-BBBB-CCCC-DDDD"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("clé invalide : %v", err)
	}
	if _, err := os.Stat(m.Path); err == nil {
		t.Fatal("fichier de licence écrit malgré une clé invalide")
	}
}

func TestCopiedInstallationDetected(t *testing.T) {
	mid := "poste-a"
	m := newManager(t, &mid)
	key := MakeKey("ABCD", "EFGH", "JKLM")
	if _, err := m.Activate(key); err != nil {
		t.Fatal(err)
	}
	// Le fichier est copié sur un autre serveur.
	mid = "poste-b"
	if s := m.Status(); s.State != StateOtherMachine {
		t.Fatalf("copie non détectée : %+v", s)
	}
	if _, err := m.Activate(key); !errors.Is(err, ErrBoundElsewhere) {
		t.Fatalf("activation sur un autre poste sans désactivation : %v", err)
	}
}

func TestTransferByDeactivateThenActivate(t *testing.T) {
	mid := "poste-a"
	m := newManager(t, &mid)
	key := MakeKey("ABCD", "EFGH", "JKLM")
	if _, err := m.Activate(key); err != nil {
		t.Fatal(err)
	}
	rec, err := m.Deactivate()
	if err != nil || rec.MachineID != "poste-a" || rec.Key != key {
		t.Fatalf("désactivation : %+v, %v", rec, err)
	}
	if _, err := m.Deactivate(); !errors.Is(err, ErrNotActivated) {
		t.Fatalf("seconde désactivation : %v", err)
	}
	mid = "poste-b"
	if s, err := m.Activate(key); err != nil || s.State != StateActive || s.MachineID != "poste-b" {
		t.Fatalf("activation sur le nouveau poste : %+v, %v", s, err)
	}
}

func TestOtherKeyNeedsDeactivation(t *testing.T) {
	mid := "poste-a"
	m := newManager(t, &mid)
	if _, err := m.Activate(MakeKey("ABCD", "EFGH", "JKLM")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Activate(MakeKey("NPQR", "STUV", "WXYZ")); !errors.Is(err, ErrOtherKey) {
		t.Fatalf("seconde licence : %v", err)
	}
}

func TestUnreadableMachineID(t *testing.T) {
	m := &Manager{Path: filepath.Join(t.TempDir(), "l.json"), Machine: func() (string, error) { return "", errors.New("illisible") }}
	if _, err := m.Activate(MakeKey("ABCD", "EFGH", "JKLM")); err == nil {
		t.Fatal("activation sans identifiant de poste")
	}
}

func TestMachineIDStableAndHashed(t *testing.T) {
	a, err := MachineID()
	if err != nil {
		t.Skipf("identifiant machine indisponible : %v", err)
	}
	b, _ := MachineID()
	if a != b || len(a) != 16 {
		t.Fatalf("identifiant instable ou mal formé : %q %q", a, b)
	}
}

func TestPathFor(t *testing.T) {
	if got := PathFor(filepath.Join("x", "config.json")); got != filepath.Join("x", "config.license.json") {
		t.Fatal(got)
	}
}

// Licence par serveur : l'ancienne licence d'un module (v1.8 et avant) est
// reprise une seule fois comme licence du serveur, puis l'ancien fichier est supprimé.
func TestLegacyModuleLicenseBecomesServerLicense(t *testing.T) {
	mid := "poste-a"
	dir := t.TempDir()
	legacy := filepath.Join(dir, "kelio", "config.license.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0755); err != nil {
		t.Fatal(err)
	}
	key := MakeKey("ABCD", "EFGH", "JKLM")
	if err := os.WriteFile(legacy, []byte(`{"key":"`+key+`","machine_id":"poste-a","activated_at":"2026-09-01T10:00:00Z"}`), 0600); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Path: filepath.Join(dir, "ProgramData", "SmartGUARD", "license.json"), LegacyPath: legacy,
		Now: func() time.Time { return t0 }, Machine: func() (string, error) { return mid, nil }}
	if s := m.Status(); s.State != StateActive {
		t.Fatalf("licence du module non reprise : %+v", s)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("ancien fichier de licence du module non supprimé")
	}
	// Un autre module du même serveur voit la même licence.
	other := &Manager{Path: m.Path, Machine: m.Machine}
	if s := other.Status(); s.State != StateActive {
		t.Fatalf("licence du serveur non partagée entre modules : %+v", s)
	}
	// Après désactivation, l'ancienne licence ne revient pas.
	if _, err := m.Deactivate(); err != nil {
		t.Fatal(err)
	}
	if s := m.Status(); s.State != StateNone {
		t.Fatalf("licence revenue après désactivation : %+v", s)
	}
}

// Installation : sans licence valide, refus ; une licence déjà active sur le
// serveur suffit pour installer d'autres modules.
func TestRequireForInstall(t *testing.T) {
	mid := "poste-a"
	m := newManager(t, &mid)
	if _, err := m.RequireForInstall(""); !errors.Is(err, ErrLicenseRequired) {
		t.Fatalf("installation sans licence : %v", err)
	}
	if _, err := m.RequireForInstall("SGRD-0000-0000-0000-0000"); err == nil {
		t.Fatal("installation avec une clé invalide acceptée")
	}
	if s := m.Status(); s.State != StateNone {
		t.Fatalf("clé invalide enregistrée : %+v", s)
	}
	key := MakeKey("ABCD", "EFGH", "JKLM")
	if s, err := m.RequireForInstall(key); err != nil || s.State != StateActive {
		t.Fatalf("installation avec clé valide : %+v %v", s, err)
	}
	if _, err := m.RequireForInstall(""); err != nil {
		t.Fatalf("second module sur un serveur déjà licencié : %v", err)
	}
	// Installation copiée sur un autre serveur : refus jusqu'à la désactivation.
	mid = "poste-b"
	if _, err := m.RequireForInstall(""); err == nil || !strings.Contains(err.Error(), "autre serveur") {
		t.Fatalf("licence d'un autre serveur acceptée : %v", err)
	}
}
