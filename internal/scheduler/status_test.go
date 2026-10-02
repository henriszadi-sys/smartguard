package scheduler

import (
	"os"
	"strings"
	"testing"
	"time"
	_ "time/tzdata" // fuseaux disponibles même sans base système (Windows)

	"smartguard/internal/config"
)

// Les tests tournent dans un fuseau à heure d'été pour couvrir les changements d'heure.
func TestMain(m *testing.M) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}
	time.Local = loc
	os.Exit(m.Run())
}

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func cfg(end string) config.Config {
	c := config.Default()
	c.Enabled = true
	c.SoftwareName = "Kelio"
	d := config.NewDeadline(config.KindContrat)
	d.ID, d.StartDate, d.EndDate = "d1", "2026-01-01", end
	c.Deadlines = []config.Deadline{d}
	return c
}

func TestExpiryAtMidnightOfEndDate(t *testing.T) {
	c := cfg("2026-10-05")
	before := ComputeStatus(c, config.State{}, at("2026-10-04 23:59:59"))
	if before.Expired || before.DaysLeft != 1 || before.Level != "critical" {
		t.Fatalf("veille à 23:59:59 : %+v", before)
	}
	s := ComputeStatus(c, config.State{}, at("2026-10-05 00:00:00"))
	if !s.Expired || s.DaysLeft != 0 || s.Level != "expired" || !s.Show {
		t.Fatalf("00:00 le jour de fin : %+v", s)
	}
}

func TestWarningLevels(t *testing.T) {
	c := cfg("2026-06-30")
	cases := []struct {
		now   string
		days  int
		show  bool
		level string
	}{
		{"2026-05-30 10:00:00", 31, false, "ok"},
		{"2026-05-31 10:00:00", 30, true, "warning"},
		{"2026-06-22 10:00:00", 8, true, "warning"},
		{"2026-06-23 10:00:00", 7, true, "critical"},
		{"2026-06-29 10:00:00", 1, true, "critical"},
	}
	for _, tc := range cases {
		s := ComputeStatus(c, config.State{}, at(tc.now))
		if s.DaysLeft != tc.days || s.Show != tc.show || s.Level != tc.level {
			t.Errorf("%s : obtenu jours=%d bandeau=%v niveau=%s, attendu %d %v %s",
				tc.now, s.DaysLeft, s.Show, s.Level, tc.days, tc.show, tc.level)
		}
	}
}

func TestMessageShowsRemainingDays(t *testing.T) {
	c := cfg("2026-06-30")
	s := ComputeStatus(c, config.State{}, at("2026-06-20 09:00:00"))
	if !strings.Contains(s.Message, "prendra fin dans 10 jours") {
		t.Fatalf("message : %q", s.Message)
	}
	s = ComputeStatus(c, config.State{}, at("2026-06-29 09:00:00"))
	if !strings.Contains(s.Message, "dans 1 jour,") {
		t.Fatalf("singulier attendu : %q", s.Message)
	}
}

func TestDisabledModuleShowsNothing(t *testing.T) {
	c := cfg("2026-06-30")
	c.Enabled = false
	if s := ComputeStatus(c, config.State{}, at("2026-06-29 09:00:00")); s.Show {
		t.Fatalf("module désactivé : bandeau affiché")
	}
}

// Le passage à l'heure d'hiver (25 octobre 2026) donne une journée de 25 h,
// celui à l'heure d'été (29 mars 2026) une journée de 23 h : le décompte doit
// rester en jours calendaires.
func TestDaysLeftAcrossDSTChange(t *testing.T) {
	if s := ComputeStatus(cfg("2026-10-27"), config.State{}, at("2026-10-24 12:00:00")); s.DaysLeft != 3 {
		t.Fatalf("heure d'hiver : %d jours, attendu 3", s.DaysLeft)
	}
	if s := ComputeStatus(cfg("2026-03-31"), config.State{}, at("2026-03-28 12:00:00")); s.DaysLeft != 3 {
		t.Fatalf("heure d'été : %d jours, attendu 3", s.DaysLeft)
	}
	// Échéance le jour même du changement d'heure : toujours à 00:00.
	if s := ComputeStatus(cfg("2026-10-25"), config.State{}, at("2026-10-25 00:00:00")); !s.Expired {
		t.Fatalf("échéance le jour du changement d'heure : %+v", s)
	}
}

func TestClockRollbackNeverPostponesExpiry(t *testing.T) {
	c := cfg("2026-10-05")
	seen := config.State{LastSeen: at("2026-10-05 00:10:00")}
	for _, back := range []string{"2026-10-05 00:05:00", "2026-10-04 23:30:00", "2026-09-01 08:00:00"} {
		if s := ComputeStatus(c, seen, at(back)); !s.Expired {
			t.Errorf("horloge reculée à %s : l'échéance est repoussée", back)
		}
	}
	if got := EffectiveNow(seen, at("2026-10-04 23:30:00")); !got.Equal(seen.LastSeen) {
		t.Errorf("EffectiveNow a reculé : %v", got)
	}
}

func TestStatusUsesServerTimeZone(t *testing.T) {
	// 23:30 UTC le 4 = 01:30 à Paris le 5 : l'échéance est atteinte.
	if s := ComputeStatus(cfg("2026-10-05"), config.State{}, time.Date(2026, 10, 4, 23, 30, 0, 0, time.UTC)); !s.Expired {
		t.Fatalf("heure UTC non convertie dans le fuseau du serveur : %+v", s)
	}
}

func TestIsBlocked(t *testing.T) {
	patterns := []string{"/kelio", "http://srv:8080/paie", "https://intranet/rh"}
	cases := []struct {
		host, path string
		want       bool
	}{
		{"srv:8080", "/kelio/accueil", true},
		{"srv:8080", "/autre", false},
		{"srv:8080", "/paie/bulletins", true},
		{"srv:9090", "/paie", false},
		{"intranet:443", "/rh", true},
		{"srv:8080", "", false},
	}
	for _, tc := range cases {
		if got := IsBlocked(patterns, tc.host, tc.path); got != tc.want {
			t.Errorf("IsBlocked(%s, %s) = %v, attendu %v", tc.host, tc.path, got, tc.want)
		}
	}
}

func TestStopOnEndFollowsContractEnd(t *testing.T) {
	c := cfg("2026-10-05")
	if s := ComputeStatus(c, config.State{}, at("2026-12-01 10:00:00")); !s.Expired || s.Stopped {
		t.Fatalf("sans option d'arrêt : %+v", s)
	}
	c.Deadlines[0].StopOnEnd = true
	if s := ComputeStatus(c, config.State{}, at("2026-10-04 23:59:59")); s.Expired || s.Stopped {
		t.Fatalf("veille de la fin : %+v", s)
	}
	if s := ComputeStatus(c, config.State{}, at("2026-10-05 00:00:00")); !s.Expired || !s.Stopped {
		t.Fatalf("00:00 le jour de la fin : %+v", s)
	}
	// Le recul d'horloge ne repousse pas l'arrêt.
	seen := config.State{LastSeen: at("2026-10-05 00:10:00")}
	if s := ComputeStatus(c, seen, at("2026-09-25 08:00:00")); !s.Stopped {
		t.Fatalf("recul d'horloge : arrêt repoussé : %+v", s)
	}
}

// Plusieurs échéances : chacune a son décompte ; le bandeau reprend la plus urgente.
func TestMultipleDeadlinesMostUrgent(t *testing.T) {
	c := cfg("2026-12-31")
	lic := config.NewDeadline(config.KindLicence)
	lic.ID, lic.EndDate = "d2", "2026-10-10"
	abo := config.NewDeadline(config.KindAbonnement)
	abo.ID, abo.EndDate, abo.StopOnEnd = "d3", "2026-10-05", false
	c.Deadlines = append(c.Deadlines, lic, abo)

	s := ComputeStatus(c, config.State{}, at("2026-10-01 10:00:00"))
	if len(s.Deadlines) != 3 || s.DeadlineID != "d3" || s.DaysLeft != 4 || s.Kind != config.KindAbonnement {
		t.Fatalf("plus urgente attendue d3 : %+v", s)
	}
	if !strings.Contains(s.Message, "abonnement") || s.Deadlines[0].Show {
		t.Fatalf("messages par type : %+v", s)
	}
	if s.Deadlines[1].DaysLeft != 9 || !s.Deadlines[1].Show || !strings.Contains(s.Deadlines[1].Message, "licence") {
		t.Fatalf("licence : %+v", s.Deadlines[1])
	}

	// Abonnement passé sans arrêt, licence arrêtée : l'arrêt prime.
	c.Deadlines[1].StopOnEnd = true
	s = ComputeStatus(c, config.State{}, at("2026-10-11 10:00:00"))
	if s.DeadlineID != "d2" || !s.Stopped || s.Deadlines[2].Stopped || !s.Deadlines[2].Expired {
		t.Fatalf("arrêt de la licence attendu en tête : %+v", s)
	}
}

// Seules les adresses des échéances arrêtées sont bloquées.
func TestBlocksOnlyStoppedDeadlines(t *testing.T) {
	c := cfg("2026-10-05")
	c.Deadlines[0].StopOnEnd = true
	c.Deadlines[0].BlockedURLs = []string{"/kelio"}
	lic := config.NewDeadline(config.KindLicence)
	lic.ID, lic.EndDate, lic.StopOnEnd, lic.BlockedURLs = "d2", "2027-01-01", true, []string{"/paie"}
	c.Deadlines = append(c.Deadlines, lic)
	s := ComputeStatus(c, config.State{}, at("2026-10-06 10:00:00"))
	if !Blocks(c, s, "srv", "/kelio/accueil") || Blocks(c, s, "srv", "/paie") {
		t.Fatalf("blocage : seul /kelio doit être bloqué")
	}
	c.Enabled = false
	if Blocks(c, ComputeStatus(c, config.State{}, at("2026-10-06 10:00:00")), "srv", "/kelio") {
		t.Fatal("module désactivé : adresse bloquée")
	}
}
