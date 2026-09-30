package scheduler

import (
	"path/filepath"
	"testing"
	"time"

	"smartguard/internal/config"
)

type fakeClock struct{ t time.Time }

func (f *fakeClock) Now() time.Time { return f.t }

func newWatcher(t *testing.T, end string, clk *fakeClock) (*Watcher, *config.Store, *int) {
	t.Helper()
	st, err := config.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *config.Config) error {
		c.Enabled, c.StartDate, c.EndDate = true, "2026-01-01", end
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	runs := 0
	return &Watcher{Store: st, Now: clk.Now, Expire: func(config.Config) { runs++ }}, st, &runs
}

func TestActionsRunOnceAtExpiry(t *testing.T) {
	clk := &fakeClock{at("2026-10-04 23:59:30")}
	w, _, runs := newWatcher(t, "2026-10-05", clk)
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
	w, st, runs := newWatcher(t, "2026-10-05", clk)
	w.Tick()
	restarted := &Watcher{Store: st, Now: clk.Now, Expire: w.Expire}
	restarted.Tick()
	if *runs != 1 {
		t.Fatalf("actions exécutées %d fois, attendu 1", *runs)
	}
	if st.State().ActionsFor != "2026-10-05" {
		t.Fatalf("échéance traitée non mémorisée : %+v", st.State())
	}
}

func TestClockRollbackAfterExpiryDoesNotRetrigger(t *testing.T) {
	clk := &fakeClock{at("2026-10-05 00:01:00")}
	w, st, runs := newWatcher(t, "2026-10-05", clk)
	w.Tick()
	clk.t = at("2026-09-20 10:00:00") // recul de l'horloge
	w.Tick()
	if *runs != 1 {
		t.Fatalf("actions exécutées %d fois, attendu 1", *runs)
	}
	if !st.State().LastSeen.Equal(at("2026-10-05 00:01:00")) {
		t.Fatalf("dernière heure vue a reculé : %v", st.State().LastSeen)
	}
	if s := ComputeStatus(st.Config(), st.State(), clk.t); !s.Expired {
		t.Fatal("le recul d'horloge a annulé l'expiration")
	}
}

// Renouvellement : la nouvelle date de fin réarme les actions pour la prochaine échéance.
func TestRenewalArmsNextExpiry(t *testing.T) {
	clk := &fakeClock{at("2026-10-05 00:00:00")}
	w, st, runs := newWatcher(t, "2026-10-05", clk)
	w.Tick()
	if err := st.UpdateConfig(func(c *config.Config) error { c.EndDate = "2027-10-05"; return nil }); err != nil {
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
	w, st, runs := newWatcher(t, "2026-10-05", clk)
	if err := st.UpdateConfig(func(c *config.Config) error { c.Enabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	w.Tick()
	if *runs != 0 {
		t.Fatal("module désactivé : actions exécutées")
	}
}
