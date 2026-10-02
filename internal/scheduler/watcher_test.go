package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"smartguard/internal/config"
)

type fakeClock struct{ t time.Time }

func (f *fakeClock) Now() time.Time { return f.t }

// Module activé, arrêt à la date de fin (stopOnEnd) ou message seul.
func newWatcher(t *testing.T, end string, stopOnEnd bool, clk *fakeClock) (*Watcher, *config.Store, *int) {
	t.Helper()
	st, err := config.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *config.Config) error {
		c.Enabled = true
		c.Deadlines = []config.Deadline{{Kind: config.KindContrat, StartDate: "2026-01-01", EndDate: end, StopOnEnd: stopOnEnd}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	runs := 0
	return &Watcher{Store: st, Now: clk.Now, Expire: func(config.Config, config.Deadline) { runs++ }}, st, &runs
}

func TestActionsRunOnceAtExpiry(t *testing.T) {
	clk := &fakeClock{at("2026-10-04 23:59:30")}
	w, _, runs := newWatcher(t, "2026-10-05", true, clk)
	w.Tick()
	if *runs != 0 {
		t.Fatal("actions exécutées avant l'échéance")
	}
	clk.t = at("2026-10-05 00:00:00")
	w.Tick()
	clk.t = clk.t.Add(30 * time.Second)
	w.Tick()
	if *runs != 1 {
		t.Fatalf("actions exécutées %d fois, attendu 1", *runs)
	}
}

// Service arrêté pendant l'échéance : au redémarrage, les actions manquées
// sont exécutées une seule fois, même si le processus redémarre encore.
func TestMissedActionsRunOnceAfterRestart(t *testing.T) {
	clk := &fakeClock{at("2026-10-09 08:00:00")}
	w, st, runs := newWatcher(t, "2026-10-05", true, clk)
	w.Tick()
	restarted := &Watcher{Store: st, Now: clk.Now, Expire: w.Expire}
	restarted.Tick()
	if *runs != 1 {
		t.Fatalf("actions exécutées %d fois, attendu 1", *runs)
	}
	if r := st.State().Actions["d1"]; r.For != "2026-10-05" || r.DoneAt.IsZero() {
		t.Fatalf("échéance traitée non mémorisée : %+v", st.State())
	}
}

func TestClockRollbackAfterExpiryDoesNotRetrigger(t *testing.T) {
	clk := &fakeClock{at("2026-10-05 00:01:00")}
	w, st, runs := newWatcher(t, "2026-10-05", true, clk)
	w.Tick()
	clk.t = at("2026-09-20 10:00:00") // recul de l'horloge
	w.Tick()
	if *runs != 1 {
		t.Fatalf("actions exécutées %d fois, attendu 1", *runs)
	}
	if !st.State().LastSeen.Equal(at("2026-10-05 00:01:00")) {
		t.Fatalf("dernière heure vue a reculé : %v", st.State().LastSeen)
	}
	if s := ComputeStatus(st.Config(), st.State(), clk.t); !s.Stopped {
		t.Fatal("le recul d'horloge a annulé l'expiration")
	}
}

// Renouvellement : la nouvelle date de fin réarme les actions pour la prochaine échéance.
func TestRenewalArmsNextExpiry(t *testing.T) {
	clk := &fakeClock{at("2026-10-05 00:00:00")}
	w, st, runs := newWatcher(t, "2026-10-05", true, clk)
	w.Tick()
	if err := st.UpdateConfig(func(c *config.Config) error { c.Deadlines[0].EndDate = "2027-10-05"; return nil }); err != nil {
		t.Fatal(err)
	}
	clk.t = at("2026-11-01 09:00:00")
	w.Tick()
	if *runs != 1 {
		t.Fatalf("renouvellement : actions relancées avant la nouvelle échéance (%d)", *runs)
	}
	clk.t = at("2027-10-05 00:00:00")
	w.Tick()
	if *runs != 2 {
		t.Fatalf("nouvelle échéance : %d exécutions, attendu 2", *runs)
	}
}

func TestDisabledModuleRunsNoAction(t *testing.T) {
	clk := &fakeClock{at("2026-10-06 00:00:00")}
	w, st, runs := newWatcher(t, "2026-10-05", true, clk)
	if err := st.UpdateConfig(func(c *config.Config) error { c.Enabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	w.Tick()
	if *runs != 0 {
		t.Fatal("module désactivé : actions exécutées")
	}
}

// Sans l'option d'arrêt, la fin du contrat n'arrête rien, même longtemps après.
func TestContractEndWithoutStopOptionRunsNoAction(t *testing.T) {
	clk := &fakeClock{at("2026-10-02 08:00:00")}
	w, st, runs := newWatcher(t, "2026-10-01", false, clk)
	w.Tick()
	clk.t = at("2027-03-01 08:00:00")
	w.Tick()
	if *runs != 0 {
		t.Fatalf("fin de contrat sans option d'arrêt : %d action(s) exécutée(s)", *runs)
	}
	if s := ComputeStatus(st.Config(), st.State(), clk.t); !s.Expired || s.Stopped {
		t.Fatalf("attendu expiré mais non arrêté : %+v", s)
	}
}

// Avec l'option d'arrêt, les actions partent à 00:00 le jour de la date de fin, une seule fois.
func TestActionsRunAtContractEnd(t *testing.T) {
	clk := &fakeClock{at("2026-10-01 00:00:00")}
	w, _, runs := newWatcher(t, "2026-11-15", true, clk)
	w.Tick()
	clk.t = at("2026-11-14 23:59:59")
	w.Tick()
	if *runs != 0 {
		t.Fatal("actions exécutées avant la date de fin")
	}
	clk.t = at("2026-11-15 00:00:00")
	w.Tick()
	w.Tick()
	if *runs != 1 {
		t.Fatalf("%d exécutions à la date de fin, attendu 1", *runs)
	}
}

// Ancien config.json : stop_date devient la date de fin de l'échéance migrée, arrêt activé.
func TestLegacyStopDateMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	legacy := `{"enabled":true,"start_date":"2026-01-01","end_date":"2026-10-01","stop_date":"2026-11-15"}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	st, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	c := st.Config()
	if len(c.Deadlines) != 1 || c.Deadlines[0].EndDate != "2026-11-15" || !c.Deadlines[0].StopOnEnd || c.StopDate != "" {
		t.Fatalf("migration incorrecte : %+v", c)
	}
}

// Chaque échéance exécute ses propres actions, une seule fois, à sa propre date.
func TestEachDeadlineRunsItsOwnActionsOnce(t *testing.T) {
	clk := &fakeClock{at("2026-10-01 08:00:00")}
	st, err := config.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *config.Config) error {
		c.Enabled = true
		c.Deadlines = []config.Deadline{
			{Kind: config.KindLicence, EndDate: "2026-10-05", StopOnEnd: true, Services: []string{"lic"}},
			{Kind: config.KindContrat, EndDate: "2026-10-10", StopOnEnd: true, Services: []string{"support"}},
			{Kind: config.KindAbonnement, EndDate: "2026-10-03", StopOnEnd: false},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var ran []string
	w := &Watcher{Store: st, Now: clk.Now, Expire: func(_ config.Config, d config.Deadline) { ran = append(ran, d.ID) }}
	w.Tick()
	clk.t = at("2026-10-05 00:00:00")
	w.Tick()
	w.Tick()
	if len(ran) != 1 || ran[0] != "d1" {
		t.Fatalf("licence seule attendue : %v", ran)
	}
	// Redémarrage après la seconde échéance : rattrapage de d2 uniquement.
	clk.t = at("2026-10-12 09:00:00")
	(&Watcher{Store: st, Now: clk.Now, Expire: w.Expire}).Tick()
	w.Tick()
	if len(ran) != 2 || ran[1] != "d2" {
		t.Fatalf("rattrapage de d2 attendu une fois : %v", ran)
	}
}
