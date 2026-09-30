package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	h, err := HashPassword("motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h, "motdepasse1") {
		t.Fatal("le mot de passe apparaît en clair dans le hash")
	}
	if !CheckPassword(h, "motdepasse1") {
		t.Fatal("bon mot de passe refusé")
	}
	if CheckPassword(h, "motdepasse2") || CheckPassword("", "motdepasse1") || CheckPassword("x$1$a$b", "motdepasse1") {
		t.Fatal("mauvais mot de passe ou hash invalide accepté")
	}
}

func TestValidate(t *testing.T) {
	c := Default()
	c.Enabled = true
	if Validate(c) == nil {
		t.Error("activation sans date de fin acceptée")
	}
	c.StartDate, c.EndDate = "2026-10-05", "2026-01-01"
	if Validate(c) == nil {
		t.Error("date de fin antérieure au début acceptée")
	}
	c.EndDate = "05/10/2026"
	if Validate(c) == nil {
		t.Error("date au mauvais format acceptée")
	}
	c.StartDate, c.EndDate = "2026-01-01", "2026-10-05"
	if err := Validate(c); err != nil {
		t.Errorf("configuration valide refusée : %v", err)
	}
}

func TestStorePersistsAndRollsBackInvalidUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *Config) error { c.SoftwareName, c.EndDate = "Kelio", "2026-10-05"; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *Config) error { c.EndDate = "invalide"; return nil }); err == nil {
		t.Fatal("mise à jour invalide acceptée")
	}
	if st.Config().EndDate != "2026-10-05" {
		t.Fatalf("configuration non restaurée après erreur : %q", st.Config().EndDate)
	}
	again, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if c := again.Config(); c.SoftwareName != "Kelio" || c.EndDate != "2026-10-05" {
		t.Fatalf("configuration relue : %+v", c)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("fichier temporaire laissé sur le disque")
	}
}

func TestDefaultsUseProductName(t *testing.T) {
	c := Default()
	if c.ModuleName != "SmartGUARD" || c.AdminPath != "/_smartguard" || c.AdminUser != "admin" || c.WarningDays != 30 {
		t.Fatalf("valeurs par défaut : %+v", c)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"Kelio Licence": "Kelio-Licence",
		"Paie.v2":       "Paie-v2",
		"Élève/École":   "lvecole",
		"***":           "SmartGUARD",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, attendu %q", in, got, want)
		}
	}
}
