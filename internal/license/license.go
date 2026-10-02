// Package license lie la licence SmartGUARD à un serveur et le signale dans l'administration.
//
// Règles du cahier des charges (section 4) :
//   - une licence par serveur (« poste »), qui couvre tous les modules de ce serveur :
//     la liaison est enregistrée une seule fois par serveur (ServerPath) ;
//   - sans licence valide, le module ne s'installe pas (contrôle fait par
//     l'assistant et la commande « install », voir RequireForInstall) ;
//   - une fois le module installé, la licence ne bloque jamais le décompte, les
//     rappels ni les actions configurées par le fournisseur ;
//   - aucun serveur central pour l'instant : la liaison est vérifiée hors ligne ;
//     une installation copiée sur une autre machine est détectée.
//
// SmartGUARD n'est pas une protection anti-piratage : ce paquet ne promet pas
// d'empêcher l'usage d'une même clé sur deux serveurs sans lien réseau.
package license

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"smartguard/internal/config"
	"smartguard/internal/platform"
)

var (
	ErrInvalidKey      = errors.New("clé de licence invalide")
	ErrBoundElsewhere  = errors.New("cette installation est liée à un autre poste : désactivez d'abord la licence")
	ErrOtherKey        = errors.New("une autre licence est déjà activée : désactivez-la d'abord")
	ErrNotActivated    = errors.New("aucune licence n'est activée")
	ErrLicenseRequired = errors.New("licence SmartGUARD obligatoire : saisissez la clé de licence de ce serveur")
)

// State : situation de la licence vis-à-vis du poste courant.
type State string

const (
	StateNone         State = "none"          // aucune licence activée
	StateActive       State = "active"        // activée sur ce poste
	StateOtherMachine State = "other_machine" // liée à un autre poste (installation copiée)
	StateError        State = "error"         // identifiant du poste illisible
)

// Record : liaison enregistrée dans license.json (clé complète : fichier en 0600).
type Record struct {
	Key         string    `json:"key"`
	MachineID   string    `json:"machine_id"`
	ActivatedAt time.Time `json:"activated_at"`
	LicenseID   string    `json:"license_id,omitempty"` // clé signée : identifiant de la licence
	Customer    string    `json:"customer,omitempty"`   // clé signée : titulaire
}

// Status : état présenté à l'administration (clé masquée).
type Status struct {
	State       State  `json:"state"`
	MaskedKey   string `json:"masked_key,omitempty"`
	MachineID   string `json:"machine_id"` // identifiant du poste courant
	ActivatedAt string `json:"activated_at,omitempty"`
	Signed      bool   `json:"signed"`             // clé signée par le fournisseur
	Customer    string `json:"customer,omitempty"` // titulaire (clé signée)
	SignedMode  bool   `json:"signed_mode"`        // ce module n'accepte que des clés signées
	Message     string `json:"message"`
}

// Manager lit et écrit la liaison licence / poste.
type Manager struct {
	Path       string                 // liaison du serveur (ServerPath)
	LegacyPath string                 // ancienne liaison propre à un module (v1.8 et avant), reprise puis supprimée
	Now        func() time.Time       // horloge injectable (time.Now par défaut)
	Machine    func() (string, error) // identifiant du poste (MachineID par défaut)
	Public     ed25519.PublicKey      // clé publique du fournisseur ; nil = mode non signé (clés SGRD-…)

	mu sync.Mutex
}

// PathFor place license.json à côté du fichier de configuration.
func PathFor(cfgPath string) string {
	return strings.TrimSuffix(cfgPath, filepath.Ext(cfgPath)) + ".license.json"
}

// ServerPath : fichier unique de licence du serveur, à côté de la liste des installations.
func ServerPath() string {
	return filepath.Join(platform.RegistryDir(), "license.json")
}

// NewManager crée le gestionnaire de la licence du serveur ; cfgPath (optionnel)
// désigne la configuration d'un module dont l'ancienne licence est reprise.
func NewManager(cfgPath string) *Manager {
	m := &Manager{Path: ServerPath(), Public: DefaultPublicKey()}
	if cfgPath != "" {
		m.LegacyPath = PathFor(cfgPath)
	}
	return m
}

// RequireForInstall : contrôle à l'installation d'un nouveau module. Renvoie nil
// si la licence du serveur est active, ou si la clé fournie l'active.
func (m *Manager) RequireForInstall(key string) (Status, error) {
	st := m.Status()
	switch st.State {
	case StateActive:
		return st, nil
	case StateOtherMachine:
		return st, errors.New("la licence SmartGUARD enregistrée est liée à un autre serveur : désactivez-la puis activez une licence sur ce serveur")
	}
	if strings.TrimSpace(key) == "" {
		return st, ErrLicenseRequired
	}
	return m.Activate(key)
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m *Manager) machine() (string, error) {
	if m.Machine != nil {
		return m.Machine()
	}
	return MachineID()
}

func (m *Manager) load() (*Record, error) {
	b, err := os.ReadFile(m.Path)
	if errors.Is(err, os.ErrNotExist) {
		return m.migrateLegacy()
	}
	if err != nil {
		return nil, err
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("fichier de licence illisible %s : %w", m.Path, err)
	}
	return &r, nil
}

// migrateLegacy reprend la licence enregistrée par un module (versions 1.8 et
// antérieures) comme licence du serveur, puis supprime l'ancien fichier.
func (m *Manager) migrateLegacy() (*Record, error) {
	if m.LegacyPath == "" {
		return nil, nil
	}
	b, err := os.ReadFile(m.LegacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil || r.Key == "" {
		return nil, nil // ancien fichier illisible : ignoré
	}
	if err := os.MkdirAll(filepath.Dir(m.Path), 0755); err != nil {
		return nil, err
	}
	if err := config.WriteJSON(m.Path, r); err != nil {
		return nil, err
	}
	_ = os.Remove(m.LegacyPath)
	return &r, nil
}

// Status calcule l'état de la licence pour le poste courant.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked()
}

func (m *Manager) statusLocked() Status {
	mid, merr := m.machine()
	s := Status{State: StateNone, MachineID: mid, SignedMode: m.Public != nil, Message: "Aucune licence SmartGUARD activée sur ce serveur. Le module installé continue de fonctionner ; une licence est exigée pour installer un nouveau module."}
	rec, err := m.load()
	if err != nil {
		s.State, s.Message = StateError, err.Error()
		return s
	}
	if rec == nil {
		return s
	}
	s.MaskedKey = Mask(rec.Key)
	s.Signed, s.Customer = IsSigned(rec.Key), rec.Customer
	s.ActivatedAt = rec.ActivatedAt.Local().Format("02/01/2006 15:04")
	switch {
	case merr != nil:
		s.State, s.Message = StateError, "Identifiant du poste illisible : "+merr.Error()
	case rec.MachineID != mid:
		s.State = StateOtherMachine
		s.Message = "Cette licence est liée à un autre poste (l'installation a été copiée). Désactivez-la puis activez-la sur ce poste."
	default:
		s.State, s.Message = StateActive, "Licence SmartGUARD activée sur ce serveur (tous les modules du serveur)."
	}
	return s
}

// Activate lie la licence au poste courant. Réactiver la même licence sur le
// même poste est sans effet.
func (m *Manager) Activate(key string) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key = NormalizeKey(key)
	var claims Claims
	if m.Public != nil { // mode signé : seules les licences émises par le fournisseur sont acceptées
		c, err := Verify(m.Public, key)
		if err != nil {
			return m.statusLocked(), err
		}
		claims = c
	} else if !CheckKey(key) {
		return m.statusLocked(), ErrInvalidKey
	}
	mid, err := m.machine()
	if err != nil {
		return m.statusLocked(), fmt.Errorf("identifiant du poste illisible : %w", err)
	}
	rec, err := m.load()
	if err != nil {
		return m.statusLocked(), err
	}
	if rec != nil {
		switch {
		case rec.MachineID != mid:
			return m.statusLocked(), ErrBoundElsewhere
		case rec.Key != key:
			return m.statusLocked(), ErrOtherKey
		default:
			return m.statusLocked(), nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(m.Path), 0755); err != nil {
		return m.statusLocked(), err
	}
	if err := config.WriteJSON(m.Path, Record{Key: key, MachineID: mid, ActivatedAt: m.now(), LicenseID: claims.ID, Customer: claims.Customer}); err != nil {
		return m.statusLocked(), err
	}
	return m.statusLocked(), nil
}

// Deactivate supprime la liaison et renvoie l'enregistrement retiré (pour le journal).
// Première étape d'un transfert : la licence peut ensuite être activée sur un autre poste.
func (m *Manager) Deactivate() (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, err := m.load()
	if err != nil {
		return Record{}, err
	}
	if rec == nil {
		return Record{}, ErrNotActivated
	}
	if err := os.Remove(m.Path); err != nil {
		return Record{}, err
	}
	return *rec, nil
}

// ---- Clé de licence : SGRD-XXXX-XXXX-XXXX-CCCC (CCCC = somme de contrôle)

const (
	keyPrefix   = "SGRD"
	keyAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // sans 0, O, 1, I
)

// NormalizeKey met la clé en majuscules et retire les espaces et retours à la ligne.
func NormalizeKey(k string) string {
	return strings.ToUpper(strings.Join(strings.Fields(k), ""))
}

func checksum(body string) string {
	sum := crc32.ChecksumIEEE([]byte(body))
	out := make([]byte, 4)
	for i := range out {
		out[i] = keyAlphabet[sum&31]
		sum >>= 5
	}
	return string(out)
}

// CheckKey vérifie le format et la somme de contrôle d'une clé normalisée.
func CheckKey(k string) bool {
	p := strings.Split(k, "-")
	if len(p) != 5 || p[0] != keyPrefix {
		return false
	}
	for _, g := range p[1:] {
		if len(g) != 4 || strings.Trim(g, keyAlphabet) != "" {
			return false
		}
	}
	return p[4] == checksum(strings.Join(p[:4], "-"))
}

// MakeKey complète trois groupes de quatre caractères par la somme de contrôle.
// Sert aux tests et à un futur outil d'émission des clés.
func MakeKey(g1, g2, g3 string) string {
	body := strings.Join([]string{keyPrefix, g1, g2, g3}, "-")
	return body + "-" + checksum(body)
}

// Mask masque une clé pour l'affichage et le journal : seuls les derniers caractères restent visibles.
func Mask(k string) string {
	if IsSigned(k) {
		if len(k) < len(signedPrefix)+6 {
			return "****"
		}
		return signedPrefix + "****" + k[len(k)-6:]
	}
	p := strings.Split(k, "-")
	if len(p) != 5 {
		return "****"
	}
	return p[0] + "-****-****-****-" + p[4]
}

// hashMachine transforme l'identifiant brut d'une machine en identifiant de poste
// (haché : l'identifiant système brut n'est ni affiché ni enregistré).
func hashMachine(raw string) string {
	h := sha256.Sum256([]byte("smartguard|" + strings.TrimSpace(raw)))
	return hex.EncodeToString(h[:])[:16]
}
