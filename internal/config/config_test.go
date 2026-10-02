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
		t.Error("activation sans échéance acceptée")
	}
	c.Deadlines = []Deadline{{ID: "d1", Kind: KindContrat, StartDate: "2026-10-05", EndDate: "2026-01-01"}}
	if Validate(c) == nil {
		t.Error("date de fin antérieure au début acceptée")
	}
	c.Deadlines[0].EndDate = "05/10/2026"
	if Validate(c) == nil {
		t.Error("date au mauvais format acceptée")
	}
	c.Deadlines[0].EndDate = ""
	if Validate(c) == nil {
		t.Error("échéance sans date de fin acceptée")
	}
	c.Deadlines[0].StartDate, c.Deadlines[0].EndDate = "2026-01-01", "2026-10-05"
	if err := Validate(c); err != nil {
		t.Errorf("configuration valide refusée : %v", err)
	}
	c.Deadlines = append(c.Deadlines, Deadline{ID: "d1", EndDate: "2027-01-01"})
	if Validate(c) == nil {
		t.Error("identifiants en double acceptés")
	}
}

func TestStorePersistsAndRollsBackInvalidUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *Config) error {
		c.SoftwareName, c.Deadlines = "Kelio", []Deadline{{Kind: KindLicence, EndDate: "2026-10-05"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *Config) error { c.Deadlines[0].EndDate = "invalide"; return nil }); err == nil {
		t.Fatal("mise à jour invalide acceptée")
	}
	if d := st.Config().Deadlines[0]; d.EndDate != "2026-10-05" || d.ID != "d1" {
		t.Fatalf("configuration non restaurée après erreur : %+v", d)
	}
	again, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if c := again.Config(); c.SoftwareName != "Kelio" || len(c.Deadlines) != 1 || c.Deadlines[0].EndDate != "2026-10-05" {
		t.Fatalf("configuration relue : %+v", c)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("fichier temporaire laissé sur le disque")
	}
}

func TestDefaultsUseProductName(t *testing.T) {
	c := Default()
	if c.ModuleName != "SmartGUARD" || c.AdminPath != "/_smartguard" || c.AdminUser != "admin" || len(c.Deadlines) != 0 {
		t.Fatalf("valeurs par défaut : %+v", c)
	}
	if d := NewDeadline(KindLicence); d.WarningDays != 30 || !strings.Contains(d.Message, "licence") {
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

func TestValidateStopOnEnd(t *testing.T) {
	c := Default()
	c.Deadlines = []Deadline{{ID: "d1", StartDate: "2026-01-01", EndDate: "2026-10-05", StopOnEnd: true}}
	if err := Validate(c); err != nil {
		t.Errorf("arrêt à la date de fin refusé : %v", err)
	}
	c.Deadlines[0].EndDate = ""
	if err := Validate(c); err == nil {
		t.Error("arrêt sans date de fin accepté")
	}
}

// Configuration v1.6 (une seule échéance, ancienne date d'arrêt) : reprise
// comme échéance « contrat de support », actions déjà exécutées conservées.
func TestMigrateLegacyConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	legacy := `{"module_name":"Kelio","software_name":"Kelio","enabled":true,
	"start_date":"2026-01-01","stop_date":"2026-10-05","warning_days":15,
	"message":"Fin dans {jours} jours","expired_message":"` + DefaultExpiredMessage + `",
	"services":["kelio"," "],"blocked_urls":["/kelio"],"scripts":[],"listen":":8080","admin_path":"/_smartguard"}`
	state := `{"last_seen":"2026-10-06T08:00:00Z","actions_done_at":"2026-10-05T00:00:10Z","actions_for":"2026-10-05"}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.state.json"), []byte(state), 0600); err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	c := st.Config()
	if len(c.Deadlines) != 1 {
		t.Fatalf("échéances après migration : %+v", c.Deadlines)
	}
	d := c.Deadlines[0]
	if d.ID != "d1" || d.Kind != KindContrat || d.StartDate != "2026-01-01" || d.EndDate != "2026-10-05" || !d.StopOnEnd ||
		d.WarningDays != 15 || d.Message != "Fin dans {jours} jours" || d.ExpiredMessage != DefaultExpiredMessage ||
		len(d.Services) != 1 || d.BlockedURLs[0] != "/kelio" {
		t.Fatalf("échéance migrée : %+v", d)
	}
	if c.EndDate != "" || c.StopDate != "" || c.Services != nil || c.Message != "" {
		t.Fatalf("anciens champs non vidés : %+v", c)
	}
	if at, ok := st.State().Done(d); !ok || at.IsZero() {
		t.Fatalf("actions déjà exécutées non reprises : %+v", st.State())
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), `"deadlines"`) || strings.Contains(string(b), `"stop_date"`) {
		t.Fatalf("configuration migrée non enregistrée : %s", b)
	}
	// Relecture : aucune nouvelle migration, même identifiant.
	again, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if c2 := again.Config(); len(c2.Deadlines) != 1 || c2.Deadlines[0].ID != "d1" {
		t.Fatalf("relecture : %+v", c2.Deadlines)
	}
	if _, ok := again.State().Done(again.Config().Deadlines[0]); !ok {
		t.Fatal("suivi des actions perdu à la relecture")
	}
}

func TestNormalizeDeadlines(t *testing.T) {
	c := Default()
	c.Deadlines = []Deadline{
		{ID: "d2", Kind: "LICENCE", EndDate: "2026-10-05", Message: DefaultMessage},
		{Kind: "inconnu", EndDate: "2026-11-05"},
		{ID: "d2", Kind: KindAbonnement, EndDate: "2026-12-05"},
	}
	normalizeConfig(&c)
	ids := []string{c.Deadlines[0].ID, c.Deadlines[1].ID, c.Deadlines[2].ID}
	if ids[0] != "d2" || ids[1] == "" || ids[2] == "" || ids[1] == ids[2] || ids[1] == "d2" || ids[2] == "d2" {
		t.Fatalf("identifiants : %v", ids)
	}
	if c.Deadlines[0].Kind != KindLicence || c.Deadlines[1].Kind != KindContrat {
		t.Fatalf("types : %+v", c.Deadlines)
	}
	if msg, _ := DefaultMessages(KindLicence); c.Deadlines[0].Message != msg {
		t.Fatalf("message par défaut non adapté au type : %q", c.Deadlines[0].Message)
	}
	if err := Validate(c); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReportLink(t *testing.T) {
	c := Default()
	if c.ReportEnabled {
		t.Fatal("le lien « Signaler un problème » doit être désactivé par défaut")
	}
	c.ReportEnabled = true
	if Validate(c) == nil {
		t.Error("lien activé sans adresse accepté")
	}
	c.ReportURL = "javascript:alert(1)"
	if Validate(c) == nil {
		t.Error("adresse non http(s) acceptée")
	}
	c.ReportURL = "https://portail.exemple.ci/signaler"
	if err := Validate(c); err != nil {
		t.Errorf("adresse valide refusée : %v", err)
	}
}
