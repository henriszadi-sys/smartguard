package scheduler

import (
	"time"

	"smartguard/internal/config"
)

// Watcher vérifie périodiquement l'échéance et exécute une seule fois les
// actions d'expiration, y compris celles manquées pendant un arrêt du service.
type Watcher struct {
	Store    *config.Store
	Now      func() time.Time      // horloge injectable (time.Now par défaut)
	Expire   func(c config.Config) // actions à l'échéance
	Interval time.Duration         // 30 s par défaut
}

func (w *Watcher) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// Run vérifie l'échéance immédiatement puis à intervalle régulier, jusqu'à stop.
func (w *Watcher) Run(stop <-chan struct{}) {
	iv := w.Interval
	if iv <= 0 {
		iv = 30 * time.Second
	}
	t := time.NewTicker(iv)
	defer t.Stop()
	w.Tick()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			w.Tick()
		}
	}
}

// Tick mémorise l'heure vue et exécute les actions si l'échéance est atteinte.
func (w *Watcher) Tick() {
	st := w.Store.State()
	now := w.now()
	if now.After(st.LastSeen) {
		w.Store.UpdateState(func(s *config.State) { s.LastSeen = now })
		st = w.Store.State()
	}
	c := w.Store.Config()
	if !c.Enabled {
		return
	}
	s := ComputeStatus(c, st, now)
	if s.Expired && (st.ActionsDoneAt.IsZero() || st.ActionsFor != c.EndDate) {
		w.Store.UpdateState(func(x *config.State) { x.ActionsDoneAt = now; x.ActionsFor = c.EndDate })
		if w.Expire != nil {
			w.Expire(c)
		}
	}
}
