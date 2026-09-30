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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const DateLayout = "2006-01-02"

// Config : paramètres saisis par l'administrateur.
type Config struct {
	ModuleName      string   `json:"module_name"`      // nom affiché du module (renommable)
	ServiceName     string   `json:"service_name"`     // nom du service système du module
	SoftwareName    string   `json:"software_name"`    // logiciel contrôlé
	SupplierContact string   `json:"supplier_contact"` // coordonnées du fournisseur (optionnel)
	Enabled         bool     `json:"enabled"`          // activer / désactiver le module
	StartDate       string   `json:"start_date"`       // AAAA-MM-JJ
	EndDate         string   `json:"end_date"`         // AAAA-MM-JJ (date d'expiration)
	WarningDays     int      `json:"warning_days"`     // début du décompte (30 = un mois)
	Message         string   `json:"message"`          // {jours} {logiciel} {date_fin}
	ExpiredMessage  string   `json:"expired_message"`
	Services        []string `json:"services"`     // services système à arrêter
	BlockedURLs     []string `json:"blocked_urls"` // URL / préfixes à bloquer
	Scripts         []string `json:"scripts"`      // commandes / scripts à exécuter

	Listen    string `json:"listen"`     // ex. ":8080"
	Upstream  string `json:"upstream"`   // ex. "http://127.0.0.1:8081" (vide = pas de proxy)
	AdminPath string `json:"admin_path"` // ex. "/_smartguard"
	TLSCert   string `json:"tls_cert"`
	TLSKey    string `json:"tls_key"`

	AdminUser         string `json:"admin_user"`
	AdminPasswordHash string `json:"admin_password_hash"`
}

// State : état interne persistant (non modifiable via l'interface).
type State struct {
	LastSeen      time.Time `json:"last_seen"`       // anti-retour d'horloge
	ActionsDoneAt time.Time `json:"actions_done_at"` // zéro = actions pas encore exécutées
	ActionsFor    string    `json:"actions_for"`     // date de fin concernée
}

const DefaultMessage = "L'assistance et le support technique à votre logiciel prendra fin dans {jours} jours, veuillez contacter le fournisseur"
const DefaultExpiredMessage = "L'assistance et le support technique à votre logiciel {logiciel} ont pris fin le {date_fin}. Veuillez contacter le fournisseur."

func Default() Config {
	return Config{
		ModuleName:     "SmartGUARD",
		ServiceName:    "SmartGUARD",
		SoftwareName:   "Mon logiciel",
		Enabled:        false,
		WarningDays:    30,
		Message:        DefaultMessage,
		ExpiredMessage: DefaultExpiredMessage,
		Services:       []string{},
		BlockedURLs:    []string{},
		Scripts:        []string{},
		Listen:         ":8080",
		Upstream:       "",
		AdminPath:      "/_smartguard",
		AdminUser:      "admin",
	}
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
	if b, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(b, &s.cfg); err != nil {
			return nil, fmt.Errorf("fichier de configuration invalide %s : %w", cfgPath, err)
		}
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
	return s, nil
}

func (s *Store) normalize() {
	c := &s.cfg
	if c.WarningDays <= 0 {
		c.WarningDays = 30
	}
	if strings.TrimSpace(c.Message) == "" {
		c.Message = DefaultMessage
	}
	if strings.TrimSpace(c.ExpiredMessage) == "" {
		c.ExpiredMessage = DefaultExpiredMessage
	}
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
	c.Services = cleanList(c.Services)
	c.BlockedURLs = cleanList(c.BlockedURLs)
	c.Scripts = cleanList(c.Scripts)
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
	return s.cfg
}

func (s *Store) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Store) UpdateConfig(fn func(c *Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cfg
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
	if c.StartDate != "" {
		if _, err := time.ParseInLocation(DateLayout, c.StartDate, time.Local); err != nil {
			return fmt.Errorf("date de début invalide (format AAAA-MM-JJ)")
		}
	}
	if c.EndDate != "" {
		if _, err := time.ParseInLocation(DateLayout, c.EndDate, time.Local); err != nil {
			return fmt.Errorf("date de fin invalide (format AAAA-MM-JJ)")
		}
	}
	if c.StartDate != "" && c.EndDate != "" && c.EndDate < c.StartDate {
		return fmt.Errorf("la date de fin doit être postérieure à la date de début")
	}
	if c.Enabled && c.EndDate == "" {
		return fmt.Errorf("renseignez la date de fin avant d'activer le module")
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
