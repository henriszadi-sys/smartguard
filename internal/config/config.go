// Package config lit, valide et enregistre la configuration d'un module
// ainsi que son état interne persistant.
package config

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const DateLayout = "2006-01-02"

// Types d'échéance suivis par un module.
const (
	KindLicence    = "licence"    // licence du logiciel
	KindContrat    = "contrat"    // contrat de support / maintenance
	KindAbonnement = "abonnement" // abonnement
)

// MaxDeadlines : nombre maximal d'échéances par module.
const MaxDeadlines = 20

// KindLabel renvoie le libellé français d'un type d'échéance.
func KindLabel(kind string) string {
	switch kind {
	case KindLicence:
		return "Licence"
	case KindAbonnement:
		return "Abonnement"
	default:
		return "Contrat de support"
	}
}

// Deadline : une échéance (licence, contrat de support ou abonnement) avec
// ses rappels, ses messages et ses actions à la date de fin.
type Deadline struct {
	ID             string   `json:"id"`              // identifiant stable (d1, d2…)
	Kind           string   `json:"kind"`            // licence | contrat | abonnement
	Label          string   `json:"label"`           // libellé libre (optionnel)
	StartDate      string   `json:"start_date"`      // AAAA-MM-JJ
	EndDate        string   `json:"end_date"`        // AAAA-MM-JJ (date de fin ; date d'arrêt si StopOnEnd)
	StopOnEnd      bool     `json:"stop_on_end"`     // exécuter les actions à 00:00 à la date de fin
	WarningDays    int      `json:"warning_days"`    // début du décompte (30 = un mois)
	Message        string   `json:"message"`         // {jours} {logiciel} {date_fin} {fournisseur}
	ExpiredMessage string   `json:"expired_message"` // message après la date de fin
	Services       []string `json:"services"`        // services système à arrêter
	BlockedURLs    []string `json:"blocked_urls"`    // URL / préfixes à bloquer
	Scripts        []string `json:"scripts"`         // commandes / scripts à exécuter
}

// Name renvoie le libellé de l'échéance, ou son type à défaut.
func (d Deadline) Name() string {
	if strings.TrimSpace(d.Label) != "" {
		return d.Label
	}
	return KindLabel(d.Kind)
}

// Config : paramètres saisis par l'administrateur.
type Config struct {
	ModuleName      string     `json:"module_name"`      // nom affiché du module (renommable)
	ServiceName     string     `json:"service_name"`     // nom du service système du module
	SoftwareName    string     `json:"software_name"`    // logiciel contrôlé
	SupplierContact string     `json:"supplier_contact"` // coordonnées du fournisseur (optionnel)
	Enabled         bool       `json:"enabled"`          // activer / désactiver le module
	Deadlines       []Deadline `json:"deadlines"`        // échéances suivies

	// Lien « Signaler un problème » dans les pages de l'application (désactivé par défaut).
	ReportEnabled bool   `json:"report_enabled"`
	ReportURL     string `json:"report_url"` // adresse de signalement (espace client), http(s)

	// Ancien format (v1.6, une seule échéance) : lu puis repris dans Deadlines et vidé.
	StartDate      string   `json:"start_date,omitempty"`
	EndDate        string   `json:"end_date,omitempty"`
	StopOnEnd      bool     `json:"stop_on_end,omitempty"`
	StopDate       string   `json:"stop_date,omitempty"`
	WarningDays    int      `json:"warning_days,omitempty"`
	Message        string   `json:"message,omitempty"`
	ExpiredMessage string   `json:"expired_message,omitempty"`
	Services       []string `json:"services,omitempty"`
	BlockedURLs    []string `json:"blocked_urls,omitempty"`
	Scripts        []string `json:"scripts,omitempty"`

	Listen    string `json:"listen"`     // ex. ":8080"
	Upstream  string `json:"upstream"`   // ex. "http://127.0.0.1:8081" (vide = pas de proxy)
	AdminPath string `json:"admin_path"` // ex. "/_smartguard"
	TLSCert   string `json:"tls_cert"`
	TLSKey    string `json:"tls_key"`

	AdminUser         string `json:"admin_user"` // technicien du fournisseur : tous les réglages
	AdminPasswordHash string `json:"admin_password_hash"`

	// Accès client (administrateur du serveur du client) : consultation et
	// réactivation des services, aucun réglage. Désactivé tant que le
	// technicien n'a pas défini de mot de passe client.
	ClientUser         string `json:"client_user"`
	ClientPasswordHash string `json:"client_password_hash,omitempty"`
}

// Deadline renvoie l'échéance d'identifiant id.
func (c Config) Deadline(id string) (Deadline, bool) {
	for _, d := range c.Deadlines {
		if d.ID == id {
			return d, true
		}
	}
	return Deadline{}, false
}

// ActionRecord : exécution des actions d'une échéance.
type ActionRecord struct {
	DoneAt     time.Time `json:"done_at"`
	For        string    `json:"for"`                  // date de fin concernée
	RestoredAt time.Time `json:"restored_at,omitzero"` // dernière réactivation explicite des services
}

// State : état interne persistant (non modifiable via l'interface).
type State struct {
	LastSeen time.Time               `json:"last_seen"`         // anti-retour d'horloge
	Actions  map[string]ActionRecord `json:"actions,omitempty"` // par identifiant d'échéance

	// Ancien format (v1.6) : repris dans Actions puis vidé.
	ActionsDoneAt time.Time `json:"actions_done_at,omitzero"`
	ActionsFor    string    `json:"actions_for,omitempty"`
}

// MarkRestored mémorise la réactivation explicite des services d'une échéance.
func (st *State) MarkRestored(id string, at time.Time) {
	r, ok := st.Actions[id]
	if !ok {
		return
	}
	r.RestoredAt = at
	st.Actions[id] = r
}

// Done indique si les actions de l'échéance d ont déjà été exécutées pour sa date de fin.
func (st State) Done(d Deadline) (time.Time, bool) {
	r, ok := st.Actions[d.ID]
	if !ok || r.DoneAt.IsZero() || r.For != d.EndDate {
		return time.Time{}, false
	}
	return r.DoneAt, true
}

const DefaultMessage = "L'assistance et le support technique à votre logiciel prendra fin dans {jours} jours, veuillez contacter le fournisseur"
const DefaultExpiredMessage = "L'assistance et le support technique à votre logiciel {logiciel} ont pris fin le {date_fin}. Veuillez contacter le fournisseur."

var defaultMessages = map[string][2]string{
	KindContrat: {DefaultMessage, DefaultExpiredMessage},
	KindLicence: {"La licence de votre logiciel {logiciel} expire dans {jours} jours, veuillez contacter le fournisseur",
		"La licence de votre logiciel {logiciel} a expiré le {date_fin}. Veuillez contacter le fournisseur."},
	KindAbonnement: {"L'abonnement à votre logiciel {logiciel} prend fin dans {jours} jours, veuillez contacter le fournisseur",
		"L'abonnement à votre logiciel {logiciel} a pris fin le {date_fin}. Veuillez contacter le fournisseur."},
}

// DefaultMessages renvoie les messages par défaut (rappel, échéance passée) d'un type d'échéance.
func DefaultMessages(kind string) (string, string) {
	m, ok := defaultMessages[kind]
	if !ok {
		m = defaultMessages[KindContrat]
	}
	return m[0], m[1]
}

func isDefaultMessage(msg string, idx int) bool {
	for _, m := range defaultMessages {
		if msg == m[idx] {
			return true
		}
	}
	return false
}

func Default() Config {
	return Config{
		ModuleName:   "SmartGUARD",
		ServiceName:  "SmartGUARD",
		SoftwareName: "Mon logiciel",
		Enabled:      false,
		Deadlines:    []Deadline{},
		Listen:       ":8080",
		Upstream:     "",
		AdminPath:    "/_smartguard",
		AdminUser:    "admin",
		ClientUser:   "client",
	}
}

// NewDeadline renvoie une échéance du type donné avec ses valeurs par défaut.
func NewDeadline(kind string) Deadline {
	d := Deadline{Kind: kind}
	normalizeDeadline(&d)
	return d
}

type Store struct {
	mu        sync.RWMutex
	cfgPath   string
	statePath string
	cfg       Config
	state     State
}

func NewStore(cfgPath string) (*Store, error) {
	s := &Store{cfgPath: cfgPath, statePath: strings.TrimSuffix(cfgPath, filepath.Ext(cfgPath)) + ".state.json"}
	s.cfg = Default()
	var raw []byte
	if b, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(b, &s.cfg); err != nil {
			return nil, fmt.Errorf("fichier de configuration invalide %s : %w", cfgPath, err)
		}
		raw = b
	} else if errors.Is(err, os.ErrNotExist) {
		if err := WriteJSON(cfgPath, s.cfg); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	if b, err := os.ReadFile(s.statePath); err == nil {
		_ = json.Unmarshal(b, &s.state)
	}
	s.normalize()
	// Ancien format repris : la configuration migrée est enregistrée.
	if raw != nil && !strings.Contains(string(raw), `"deadlines"`) {
		if err := WriteJSON(cfgPath, s.cfg); err != nil {
			return nil, err
		}
		_ = WriteJSON(s.statePath, s.state)
	}
	return s, nil
}

func (s *Store) normalize() {
	normalizeConfig(&s.cfg)
	migrateState(&s.state, s.cfg)
}

func normalizeConfig(c *Config) {
	migrateLegacy(c)
	if c.Deadlines == nil {
		c.Deadlines = []Deadline{}
	}
	used := map[string]bool{}
	for i := range c.Deadlines {
		d := &c.Deadlines[i]
		d.ID = strings.TrimSpace(d.ID)
		if d.ID != "" && used[d.ID] {
			d.ID = "" // doublon : nouvel identifiant
		}
		if d.ID != "" {
			used[d.ID] = true
		}
	}
	n := 0
	for i := range c.Deadlines {
		d := &c.Deadlines[i]
		for d.ID == "" {
			n++
			if id := fmt.Sprintf("d%d", n); !used[id] {
				d.ID, used[id] = id, true
			}
		}
		normalizeDeadline(d)
	}
	c.ReportURL = strings.TrimSpace(c.ReportURL)
	if c.AdminPath == "" {
		c.AdminPath = "/_smartguard"
	}
	if !strings.HasPrefix(c.AdminPath, "/") {
		c.AdminPath = "/" + c.AdminPath
	}
	c.AdminPath = strings.TrimRight(c.AdminPath, "/")
	if c.AdminUser == "" {
		c.AdminUser = "admin"
	}
	if c.ClientUser = strings.TrimSpace(c.ClientUser); c.ClientUser == "" {
		c.ClientUser = "client"
	}
}

func normalizeDeadline(d *Deadline) {
	d.Kind = strings.ToLower(strings.TrimSpace(d.Kind))
	if _, ok := defaultMessages[d.Kind]; !ok {
		d.Kind = KindContrat
	}
	d.Label = strings.TrimSpace(d.Label)
	d.StartDate, d.EndDate = strings.TrimSpace(d.StartDate), strings.TrimSpace(d.EndDate)
	if d.WarningDays <= 0 {
		d.WarningDays = 30
	}
	msg, exp := DefaultMessages(d.Kind)
	// Message vide ou message par défaut d'un autre type : message par défaut du type.
	if strings.TrimSpace(d.Message) == "" || isDefaultMessage(d.Message, 0) {
		d.Message = msg
	}
	if strings.TrimSpace(d.ExpiredMessage) == "" || isDefaultMessage(d.ExpiredMessage, 1) {
		d.ExpiredMessage = exp
	}
	d.Services = cleanList(d.Services)
	d.BlockedURLs = cleanList(d.BlockedURLs)
	d.Scripts = cleanList(d.Scripts)
}

// migrateLegacy reprend l'échéance unique de l'ancien format (v1.6) comme
// première échéance, de type « contrat de support », puis vide les anciens champs.
func migrateLegacy(c *Config) {
	// L'ancienne date d'arrêt devient la date de fin, avec arrêt activé.
	if c.StopDate != "" {
		c.EndDate, c.StopOnEnd = c.StopDate, true
	}
	legacy := c.StartDate != "" || c.EndDate != "" || len(cleanList(c.Services)) > 0 ||
		len(cleanList(c.BlockedURLs)) > 0 || len(cleanList(c.Scripts)) > 0
	if legacy && len(c.Deadlines) == 0 {
		c.Deadlines = []Deadline{{
			Kind: KindContrat, StartDate: c.StartDate, EndDate: c.EndDate, StopOnEnd: c.StopOnEnd,
			WarningDays: c.WarningDays, Message: c.Message, ExpiredMessage: c.ExpiredMessage,
			Services: c.Services, BlockedURLs: c.BlockedURLs, Scripts: c.Scripts,
		}}
	}
	c.StartDate, c.EndDate, c.StopOnEnd, c.StopDate = "", "", false, ""
	c.WarningDays, c.Message, c.ExpiredMessage = 0, "", ""
	c.Services, c.BlockedURLs, c.Scripts = nil, nil, nil
}

// migrateState reprend l'ancien suivi des actions (une seule échéance) pour
// l'échéance de même date de fin, afin de ne pas réexécuter les actions.
func migrateState(st *State, c Config) {
	if st.ActionsDoneAt.IsZero() && st.ActionsFor == "" {
		return
	}
	for _, d := range c.Deadlines {
		if d.EndDate == st.ActionsFor && !st.ActionsDoneAt.IsZero() {
			if _, ok := st.Actions[d.ID]; !ok {
				if st.Actions == nil {
					st.Actions = map[string]ActionRecord{}
				}
				st.Actions[d.ID] = ActionRecord{DoneAt: st.ActionsDoneAt, For: st.ActionsFor}
			}
			break
		}
	}
	st.ActionsDoneAt, st.ActionsFor = time.Time{}, ""
}

func cleanList(in []string) []string {
	out := []string{}
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (s *Store) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Clone()
}

// Clone renvoie une copie indépendante (listes comprises) de la configuration.
func (c Config) Clone() Config {
	if c.Deadlines != nil {
		ds := make([]Deadline, len(c.Deadlines))
		for i, d := range c.Deadlines {
			d.Services = slices.Clone(d.Services)
			d.BlockedURLs = slices.Clone(d.BlockedURLs)
			d.Scripts = slices.Clone(d.Scripts)
			ds[i] = d
		}
		c.Deadlines = ds
	}
	c.Services, c.BlockedURLs, c.Scripts = slices.Clone(c.Services), slices.Clone(c.BlockedURLs), slices.Clone(c.Scripts)
	return c
}

func (s *Store) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.state
	st.Actions = maps.Clone(st.Actions)
	return st
}

func (s *Store) UpdateConfig(fn func(c *Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cfg.Clone()
	if err := fn(&c); err != nil {
		return err
	}
	old := s.cfg
	s.cfg = c
	s.normalize()
	if err := Validate(s.cfg); err != nil {
		s.cfg = old
		return err
	}
	return WriteJSON(s.cfgPath, s.cfg)
}

func (s *Store) UpdateState(fn func(st *State)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.state)
	_ = WriteJSON(s.statePath, s.state)
}

func Validate(c Config) error {
	if len(c.Deadlines) > MaxDeadlines {
		return fmt.Errorf("%d échéances au maximum par module", MaxDeadlines)
	}
	ids := map[string]bool{}
	for i, d := range c.Deadlines {
		if err := ValidateDeadline(d); err != nil {
			return fmt.Errorf("échéance %d (%s) : %w", i+1, d.Name(), err)
		}
		if d.ID != "" && ids[d.ID] {
			return fmt.Errorf("identifiant d'échéance en double : %s", d.ID)
		}
		ids[d.ID] = true
	}
	if c.ReportURL != "" {
		if u, err := url.Parse(c.ReportURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("adresse de signalement invalide (ex. https://portail.exemple.ci/signaler)")
		}
	}
	if c.ReportEnabled && c.ReportURL == "" {
		return fmt.Errorf("renseignez l'adresse de signalement pour activer le lien « Signaler un problème »")
	}
	if strings.EqualFold(c.ClientUser, c.AdminUser) {
		return fmt.Errorf("l'identifiant de l'accès client doit être différent de celui du technicien")
	}
	if c.Enabled && len(c.Deadlines) == 0 {
		return fmt.Errorf("ajoutez au moins une échéance avant d'activer le module")
	}
	return nil
}

// ValidateDeadline vérifie les dates d'une échéance.
func ValidateDeadline(d Deadline) error {
	if d.StartDate != "" {
		if _, err := time.ParseInLocation(DateLayout, d.StartDate, time.Local); err != nil {
			return fmt.Errorf("date de début invalide (format AAAA-MM-JJ)")
		}
	}
	if d.EndDate == "" {
		return fmt.Errorf("renseignez la date de fin")
	}
	if _, err := time.ParseInLocation(DateLayout, d.EndDate, time.Local); err != nil {
		return fmt.Errorf("date de fin invalide (format AAAA-MM-JJ)")
	}
	if d.StartDate != "" && d.EndDate < d.StartDate {
		return fmt.Errorf("la date de fin doit être postérieure à la date de début")
	}
	return nil
}

// WriteJSON écrit un fichier JSON de façon atomique (fichier temporaire puis renommage).
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---- Mot de passe administrateur (PBKDF2-SHA256) ----

func HashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const iter = 210000
	key, err := pbkdf2.Key(sha256.New, pw, salt, iter, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iter,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func CheckPassword(hash, pw string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	var iter int
	if _, err := fmt.Sscanf(parts[1], "%d", &iter); err != nil || iter < 1000 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// OpenExisting lit une configuration existante sans en créer.
func OpenExisting(path string) (*Store, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return NewStore(path)
}

// SanitizeName transforme un nom de module en nom de service système valide.
func SanitizeName(n string) string {
	var b strings.Builder
	for _, r := range n {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "SmartGUARD"
	}
	return b.String()
}
