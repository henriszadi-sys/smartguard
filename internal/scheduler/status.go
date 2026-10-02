// Package scheduler calcule l'état du décompte (rappel, expiration) et
// déclenche une seule fois les actions d'échéance.
package scheduler

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"smartguard/internal/config"
)

// DeadlineStatus : état d'une échéance à un instant donné.
type DeadlineStatus struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`       // licence | contrat | abonnement
	KindLabel     string `json:"kind_label"` // libellé du type
	Label         string `json:"label"`      // libellé affiché (libellé libre ou type)
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	StopOnEnd     bool   `json:"stop_on_end"`
	Configured    bool   `json:"configured"`
	DaysLeft      int    `json:"days_left"`
	Show          bool   `json:"show"`
	Expired       bool   `json:"expired"`
	Stopped       bool   `json:"stopped"`
	Level         string `json:"level"`
	Message       string `json:"message"`
	Progress      int    `json:"progress"`
	ExpiresAt     string `json:"expires_at"`
	ActionsDoneAt string `json:"actions_done_at,omitempty"` // dernier arrêt exécuté, JJ/MM/AAAA HH:MM
	ActionsFor    string `json:"actions_for,omitempty"`     // date de fin à l'origine de ce dernier arrêt
	RestoredAt    string `json:"restored_at,omitempty"`     // dernière réactivation des services
	CanRestore    bool   `json:"can_restore"`               // arrêt passé, échéance renouvelée, services pas encore réactivés
}

// Status : état du module. Les champs de premier niveau reprennent l'échéance
// la plus urgente (bandeau) ; Deadlines détaille chaque échéance.
type Status struct {
	ModuleName   string           `json:"module_name"`
	SoftwareName string           `json:"software_name"`
	Enabled      bool             `json:"enabled"`
	Configured   bool             `json:"configured"`
	DeadlineID   string           `json:"deadline_id"` // échéance la plus urgente
	Kind         string           `json:"kind"`
	Label        string           `json:"label"`
	StartDate    string           `json:"start_date"`
	EndDate      string           `json:"end_date"`
	StopOnEnd    bool             `json:"stop_on_end"` // arrêt complet prévu à la date de fin
	DaysLeft     int              `json:"days_left"`   // jours restants avant la date de fin
	Show         bool             `json:"show"`        // afficher le bandeau
	Expired      bool             `json:"expired"`     // date de fin atteinte
	Stopped      bool             `json:"stopped"`     // au moins une échéance arrêtée : actions et blocage
	Level        string           `json:"level"`       // ok | warning | critical | expired
	Message      string           `json:"message"`
	Contact      string           `json:"contact"`
	Progress     int              `json:"progress"` // % de la période écoulée
	ExpiresAt    string           `json:"expires_at"`
	Blocked      bool             `json:"blocked,omitempty"`
	BlockedBy    string           `json:"blocked_by,omitempty"` // libellé de l'échéance qui bloque la page
	ReportURL    string           `json:"report_url,omitempty"` // lien « Signaler un problème » (si activé)
	Deadlines    []DeadlineStatus `json:"deadlines"`
}

// EffectiveNow protège contre un retour en arrière de l'horloge système :
// le temps pris en compte ne recule jamais sous la dernière heure vue.
func EffectiveNow(st config.State, now time.Time) time.Time {
	if now.Before(st.LastSeen) {
		return st.LastSeen
	}
	return now
}

func frDate(t time.Time) string { return t.Format("02/01/2006") }

// ComputeStatus calcule l'état de chaque échéance à l'instant now et
// retient la plus urgente pour le bandeau.
func ComputeStatus(c config.Config, st config.State, now time.Time) Status {
	s := Status{ModuleName: c.ModuleName, SoftwareName: c.SoftwareName, Enabled: c.Enabled,
		Contact: c.SupplierContact, Level: "ok", Deadlines: []DeadlineStatus{}}
	if c.ReportEnabled {
		s.ReportURL = c.ReportURL
	}
	best := -1
	for _, d := range c.Deadlines {
		ds := ComputeDeadline(c, d, st, now)
		s.Deadlines = append(s.Deadlines, ds)
		if ds.Configured && (best < 0 || moreUrgent(ds, s.Deadlines[best])) {
			best = len(s.Deadlines) - 1
		}
	}
	if best < 0 {
		return s
	}
	d := s.Deadlines[best]
	s.Configured, s.DeadlineID, s.Kind, s.Label = true, d.ID, d.Kind, d.Label
	s.StartDate, s.EndDate, s.StopOnEnd = d.StartDate, d.EndDate, d.StopOnEnd
	s.DaysLeft, s.Show, s.Expired, s.Stopped = d.DaysLeft, d.Show, d.Expired, d.Stopped
	s.Level, s.Message, s.Progress, s.ExpiresAt = d.Level, d.Message, d.Progress, d.ExpiresAt
	return s
}

// moreUrgent : échéance arrêtée, puis échéance passée, puis la plus proche.
func moreUrgent(a, b DeadlineStatus) bool {
	rank := func(d DeadlineStatus) int {
		switch {
		case d.Stopped:
			return 0
		case d.Expired:
			return 1
		}
		return 2
	}
	if ra, rb := rank(a), rank(b); ra != rb {
		return ra < rb
	}
	return a.DaysLeft < b.DaysLeft
}

// ComputeDeadline calcule l'état d'une échéance à l'instant now.
func ComputeDeadline(c config.Config, d config.Deadline, st config.State, now time.Time) DeadlineStatus {
	s := DeadlineStatus{ID: d.ID, Kind: d.Kind, KindLabel: config.KindLabel(d.Kind), Label: d.Name(),
		StartDate: d.StartDate, EndDate: d.EndDate, StopOnEnd: d.StopOnEnd, Level: "ok"}
	// Dernier arrêt exécuté, conservé après renouvellement pour la réactivation.
	r, hasRecord := st.Actions[d.ID]
	if hasRecord && !r.DoneAt.IsZero() {
		s.ActionsDoneAt = r.DoneAt.In(time.Local).Format("02/01/2006 15:04")
		s.ActionsFor = r.For
		if !r.RestoredAt.IsZero() {
			s.RestoredAt = r.RestoredAt.In(time.Local).Format("02/01/2006 15:04")
		}
	}
	end, err := time.ParseInLocation(config.DateLayout, d.EndDate, time.Local)
	if err != nil {
		return s
	}
	s.Configured = true
	stop := end // échéance le jour de la date de fin, à 00:00 (heure du serveur)
	s.ExpiresAt = stop.Format(time.RFC3339)
	now = EffectiveNow(st, now).In(time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	s.DaysLeft = int(math.Round(stop.Sub(today).Hours() / 24))
	s.Expired = !now.Before(stop)
	// L'arrêt complet a lieu à 00:00 à la date de fin, uniquement si l'option d'arrêt est activée.
	s.Stopped = s.Expired && d.StopOnEnd
	// Réactivation proposée : arrêt passé, échéance plus arrêtée (renouvelée ou module
	// désactivé), services listés et pas encore réactivés depuis cet arrêt.
	s.CanRestore = hasRecord && !r.DoneAt.IsZero() && !(s.Stopped && c.Enabled) && len(d.Services) > 0 && r.RestoredAt.Before(r.DoneAt)

	if start, err := time.ParseInLocation(config.DateLayout, d.StartDate, time.Local); err == nil && stop.After(start) {
		p := now.Sub(start).Seconds() / stop.Sub(start).Seconds() * 100
		s.Progress = int(math.Max(0, math.Min(100, p)))
	}

	repl := strings.NewReplacer("{logiciel}", c.SoftwareName, "{date_fin}", frDate(end),
		"{fournisseur}", c.SupplierContact)
	switch {
	case s.Expired:
		s.DaysLeft = 0
		s.Level = "expired"
		s.Show = true
		s.Message = repl.Replace(strings.ReplaceAll(d.ExpiredMessage, "{jours}", "0"))
	case s.DaysLeft <= d.WarningDays:
		s.Show = true
		s.Level = "warning"
		if s.DaysLeft <= 7 {
			s.Level = "critical"
		}
		msg := strings.ReplaceAll(d.Message, "{jours}", fmt.Sprint(s.DaysLeft))
		if s.DaysLeft == 1 {
			msg = strings.ReplaceAll(msg, "1 jours", "1 jour")
		}
		s.Message = repl.Replace(msg)
	}
	if !c.Enabled {
		s.Show = false
	}
	return s
}

// Blocks indique si l'adresse demandée est bloquée par une échéance arrêtée
// (module activé uniquement).
func Blocks(c config.Config, s Status, host, path string) bool {
	_, ok := BlockingDeadline(c, s, host, path)
	return ok
}

// BlockingDeadline renvoie l'échéance arrêtée qui bloque l'adresse demandée.
func BlockingDeadline(c config.Config, s Status, host, path string) (DeadlineStatus, bool) {
	if !c.Enabled || !s.Stopped {
		return DeadlineStatus{}, false
	}
	for _, ds := range s.Deadlines {
		if !ds.Stopped {
			continue
		}
		if d, ok := c.Deadline(ds.ID); ok && IsBlocked(d.BlockedURLs, host, path) {
			return ds, true
		}
	}
	return DeadlineStatus{}, false
}

// ForBlockedPage adapte l'état à la page bloquée : message et libellé de
// l'échéance à l'origine du blocage.
func (s Status) ForBlockedPage(ds DeadlineStatus) Status {
	s.Blocked, s.BlockedBy = true, ds.Label
	s.DeadlineID, s.Kind, s.Label, s.Message = ds.ID, ds.Kind, ds.Label, ds.Message
	s.EndDate, s.Level, s.Expired, s.DaysLeft = ds.EndDate, ds.Level, ds.Expired, ds.DaysLeft
	return s
}

// Public renvoie l'état publié aux pages de l'application (bandeau) : sans
// détail d'exécution des actions, avec les seules échéances à afficher.
func (s Status) Public() Status {
	pub := s
	pub.Deadlines = []DeadlineStatus{}
	for _, d := range s.Deadlines {
		if d.Show {
			d.ActionsDoneAt, d.ActionsFor, d.RestoredAt, d.CanRestore = "", "", "", false
			pub.Deadlines = append(pub.Deadlines, d)
		}
	}
	return pub
}

// IsBlocked indique si l'URL demandée correspond à l'un des motifs à bloquer.
// Motifs acceptés : chemin ("/", "/kelio") ou URL complète ("http://srv:8080/kelio").
func IsBlocked(patterns []string, host, path string) bool {
	if path == "" {
		path = "/"
	}
	for _, pat := range patterns {
		if strings.HasPrefix(pat, "http://") || strings.HasPrefix(pat, "https://") {
			u, err := url.Parse(pat)
			if err != nil {
				continue
			}
			p := u.Path
			if p == "" {
				p = "/"
			}
			if strings.EqualFold(u.Host, host) && strings.HasPrefix(path, p) {
				return true
			}
			// hôte sans port indiqué : on compare le nom seul
			if !strings.Contains(u.Host, ":") && strings.EqualFold(u.Host, HostOnly(host)) && strings.HasPrefix(path, p) {
				return true
			}
			continue
		}
		if !strings.HasPrefix(pat, "/") {
			pat = "/" + pat
		}
		if strings.HasPrefix(path, pat) {
			return true
		}
	}
	return false
}

// HostOnly retire le port d'une adresse « hôte:port ».
func HostOnly(h string) string {
	if i := strings.LastIndex(h, ":"); i > 0 && !strings.Contains(h[i:], "]") {
		return h[:i]
	}
	return h
}
