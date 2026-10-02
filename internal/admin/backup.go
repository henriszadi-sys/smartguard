package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"smartguard/internal/config"
	"smartguard/internal/version"
)

// Export : fichier de sauvegarde de la configuration d'un module (réglage réservé
// au technicien). Il ne contient aucun mot de passe ni haché de mot de passe.
type Export struct {
	Format     string        `json:"format"` // "smartguard-config-v1"
	Version    string        `json:"version"`
	ExportedAt time.Time     `json:"exported_at"`
	Config     config.Config `json:"config"`
}

const exportFormat = "smartguard-config-v1"

// ExportOf prépare la sauvegarde d'une configuration, sans secrets.
func ExportOf(c config.Config, now time.Time) Export {
	c = c.Clone()
	c.AdminPasswordHash, c.ClientPasswordHash = "", ""
	return Export{Format: exportFormat, Version: version.Number, ExportedAt: now, Config: c}
}

// ApplyImport reprend d'une sauvegarde les réglages fonctionnels du module :
// noms, contact, activation, échéances et lien de signalement. Le port, l'adresse
// de l'application, le chemin d'administration, le HTTPS et les mots de passe
// restent ceux du module en place.
func ApplyImport(dst *config.Config, e Export) error {
	if e.Format != exportFormat {
		return fmt.Errorf("fichier de sauvegarde SmartGUARD non reconnu")
	}
	src := e.Config.Clone()
	if strings.TrimSpace(src.ModuleName) != "" {
		dst.ModuleName = strings.TrimSpace(src.ModuleName)
	}
	dst.SoftwareName = strings.TrimSpace(src.SoftwareName)
	dst.SupplierContact = strings.TrimSpace(src.SupplierContact)
	dst.Enabled = src.Enabled
	dst.Deadlines = src.Deadlines
	// Ancien format (v1.6) dans une sauvegarde : migré par la normalisation.
	dst.StartDate, dst.EndDate, dst.StopOnEnd, dst.StopDate = src.StartDate, src.EndDate, src.StopOnEnd, src.StopDate
	dst.WarningDays, dst.Message, dst.ExpiredMessage = src.WarningDays, src.Message, src.ExpiredMessage
	dst.Services, dst.BlockedURLs, dst.Scripts = src.Services, src.BlockedURLs, src.Scripts
	dst.ReportEnabled, dst.ReportURL = src.ReportEnabled, src.ReportURL
	return nil
}

func (s *Server) apiExport(w http.ResponseWriter, r *http.Request, c config.Config) {
	now := s.now()
	name := config.SanitizeName(c.ModuleName) + "-" + now.Format("20060102-1504") + ".smartguard.json"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(ExportOf(c, now))
	s.Log.Printf("Configuration exportée par le technicien (%s)", clientIP(r))
}

func (s *Server) apiImport(w http.ResponseWriter, r *http.Request) {
	var e Export
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&e); err != nil {
		jsonError(w, "fichier de sauvegarde illisible")
		return
	}
	err := s.Store.UpdateConfig(func(x *config.Config) error { return ApplyImport(x, e) })
	if err != nil {
		jsonError(w, err.Error())
		return
	}
	after := s.Store.Config()
	s.Log.Printf("Configuration importée par le technicien (%s) : sauvegarde du %s (v%s), %d échéance(s)",
		clientIP(r), e.ExportedAt.Local().Format("02/01/2006 15:04"), e.Version, len(after.Deadlines))
	if s.Changed != nil {
		go s.Changed()
	}
	WriteJSON(w, map[string]any{"ok": true, "deadlines": len(after.Deadlines)})
}
