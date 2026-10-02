package scheduler

import (
	"time"

	"smartguard/internal/config"
)

// Watcher vérifie périodiquement l'échéance et exécute une seule fois les
// actions d'arrêt, y compris celles manquées pendant un arrêt du service.
type Watcher struct {
	Store    *config.Store
	Now      func() time.Time                         // horloge injectable (time.Now par défaut)
	Expire   func(c config.Config, d config.Deadline) // actions d'une échéance à sa date d'arrêt
	Interval time.Duration                            // 30 s par défaut
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

// Tick mémorise l'heure vue et exécute, une seule fois par date de fin,
// les actions de chaque échéance arrêtée.
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
	for _, d := range c.Deadlines {
		if _, done := st.Done(d); done || !ComputeDeadline(c, d, st, now).Stopped {
			continue
		}
		w.Store.UpdateState(func(x *config.State) {
			if x.Actions == nil {
				x.Actions = map[string]config.ActionRecord{}
			}
			x.Actions[d.ID] = config.ActionRecord{DoneAt: now, For: d.EndDate}
		})
		if w.Expire != nil {
			w.Expire(c, d)
		}
	}
}
