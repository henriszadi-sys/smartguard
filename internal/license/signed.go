package license

import (
	"crypto/ed25519"
	"crypto/rand"
	_ "embed"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Clés signées : SGL1.<base32(charge utile ‖ signature Ed25519)>.
//
// Le fournisseur signe la charge utile avec sa clé privée (outil cmd/sgkeys) ;
// le module ne contient que la clé publique (public.key, intégrée au binaire).
// Sans clé publique, le module reste en mode « non signé » (clés SGRD-…).
//
// Charge utile : version (1 octet) ‖ identifiant (8) ‖ date d'émission (4, secondes Unix)
// ‖ longueur du titulaire (1) ‖ titulaire (UTF-8, 60 octets max).
// Comme pour tout programme distribué, la clé publique peut être remplacée par quelqu'un
// qui modifie le binaire : ce n'est pas une protection anti-piratage.

const (
	signedPrefix  = "SGL1."
	signedVersion = 1
	maxCustomer   = 60
	sigDomain     = "smartguard-license-v1\x00"
)

var (
	ErrBadSignature = errors.New("clé de licence non émise par le fournisseur (signature invalide)")
	b32             = base32.StdEncoding.WithPadding(base32.NoPadding)
)

//go:embed public.key
var embeddedPublicKey string

// Claims : contenu signé d'une licence.
type Claims struct {
	ID       string    // identifiant de la licence (16 caractères hexadécimaux)
	Customer string    // titulaire (optionnel)
	Issued   time.Time // date d'émission
}

// DefaultPublicKey renvoie la clé publique intégrée au binaire (nil = mode non signé).
func DefaultPublicKey() ed25519.PublicKey {
	k, err := ParsePublicKey(embeddedPublicKey)
	if err != nil {
		return nil
	}
	return k
}

// ParsePublicKey décode une clé publique Ed25519 en base64 ; vide = nil (mode non signé).
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, errors.New("clé publique invalide")
	}
	return ed25519.PublicKey(b), nil
}

// GenerateKeyPair crée une paire de clés du fournisseur.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// IsSigned indique si la clé a la forme d'une clé signée.
func IsSigned(key string) bool { return strings.HasPrefix(key, signedPrefix) }

// Sign émet une licence signée. L'identifiant est tiré au hasard si Claims.ID est vide.
func Sign(priv ed25519.PrivateKey, c Claims) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", errors.New("clé privée invalide")
	}
	id := make([]byte, 8)
	if c.ID == "" {
		if _, err := rand.Read(id); err != nil {
			return "", err
		}
	} else if b, err := hex.DecodeString(c.ID); err != nil || len(b) != 8 {
		return "", errors.New("identifiant de licence invalide (16 caractères hexadécimaux)")
	} else {
		id = b
	}
	cust := strings.TrimSpace(c.Customer)
	if len(cust) > maxCustomer {
		return "", fmt.Errorf("titulaire trop long (%d octets maximum)", maxCustomer)
	}
	issued := c.Issued
	if issued.IsZero() {
		issued = time.Now()
	}
	payload := make([]byte, 0, 14+len(cust))
	payload = append(payload, signedVersion)
	payload = append(payload, id...)
	payload = binary.BigEndian.AppendUint32(payload, uint32(issued.Unix()))
	payload = append(payload, byte(len(cust)))
	payload = append(payload, cust...)
	sig := ed25519.Sign(priv, append([]byte(sigDomain), payload...))
	return signedPrefix + b32.EncodeToString(append(payload, sig...)), nil
}

// Verify contrôle la signature d'une clé normalisée et renvoie son contenu.
func Verify(pub ed25519.PublicKey, key string) (Claims, error) {
	if !IsSigned(key) {
		return Claims{}, ErrInvalidKey
	}
	raw, err := b32.DecodeString(strings.TrimPrefix(key, signedPrefix))
	if err != nil || len(raw) < 14+ed25519.SignatureSize {
		return Claims{}, ErrInvalidKey
	}
	payload, sig := raw[:len(raw)-ed25519.SignatureSize], raw[len(raw)-ed25519.SignatureSize:]
	if payload[0] != signedVersion || len(payload) != 14+int(payload[13]) {
		return Claims{}, ErrInvalidKey
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, append([]byte(sigDomain), payload...), sig) {
		return Claims{}, ErrBadSignature
	}
	return Claims{
		ID:       hex.EncodeToString(payload[1:9]),
		Customer: string(payload[14:]),
		Issued:   time.Unix(int64(binary.BigEndian.Uint32(payload[9:13])), 0),
	}, nil
}
