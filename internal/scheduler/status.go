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

type Status struct {
	ModuleName   string `json:"module_name"`
	SoftwareName string `json:"software_name"`
	Enabled      bool   `json:"enabled"`
	Configured   bool   `json:"configured"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	StopOnEnd    bool   `json:"stop_on_end"` // arrêt complet prévu à la date de fin
	DaysLeft     int    `json:"days_left"`   // jours restants avant la date de fin
	Show         bool   `json:"show"`        // afficher le bandeau
	Expired      bool   `json:"expired"`     // date de fin atteinte
	Stopped      bool   `json:"stopped"`     // date de fin atteinte et arrêt activé : actions et blocage
	Level        string `json:"level"`       // ok | warning | critical | expired
	Message      string `json:"message"`
	Contact      string `json:"contact"`
	Progress     int    `json:"progress"` // % du contrat écoulé
	ExpiresAt    string `json:"expires_at"`
	Blocked      bool   `json:"blocked,omitempty"`
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

// ComputeStatus calcule l'état du décompte à l'instant now.
func ComputeStatus(c config.Config, st config.State, now time.Time) Status {
	s := Status{ModuleName: c.ModuleName, SoftwareName: c.SoftwareName, Enabled: c.Enabled,
		StartDate: c.StartDate, EndDate: c.EndDate, StopOnEnd: c.StopOnEnd, Contact: c.SupplierContact, Level: "ok"}
	end, err := time.ParseInLocation(config.DateLayout, c.EndDate, time.Local)
	if err != nil {
		return s
	}
	s.Configured = true
	stop := end // expiration (arrêt des services) le jour de la date de fin, à 00:00
	s.ExpiresAt = stop.Format(time.RFC3339)
	now = EffectiveNow(st, now).In(time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	s.DaysLeft = int(math.Round(stop.Sub(today).Hours() / 24))
	s.Expired = !now.Before(stop)
	// L'arrêt complet a lieu à 00:00 à la date de fin, uniquement si l'option d'arrêt est activée.
	s.Stopped = s.Expired && c.StopOnEnd

	if start, err := time.ParseInLocation(config.DateLayout, c.StartDate, time.Local); err == nil && stop.After(start) {
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
		s.Message = repl.Replace(strings.ReplaceAll(c.ExpiredMessage, "{jours}", "0"))
	case s.DaysLeft <= c.WarningDays:
		s.Show = true
		s.Level = "warning"
		if s.DaysLeft <= 7 {
			s.Level = "critical"
		}
		msg := strings.ReplaceAll(c.Message, "{jours}", fmt.Sprint(s.DaysLeft))
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

// IsBlocked indique si l'URL demandée fait partie des URL à bloquer après expiration.
// Motifs acceptés : chemin ("/", "/kelio") ou URL complète ("http://srv:8080/kelio").
func IsBlocked(c config.Config, host, path string) bool {
	if path == "" {
		path = "/"
	}
	for _, pat := range c.BlockedURLs {
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
