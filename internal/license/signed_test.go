package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func signedManager(t *testing.T, pub ed25519.PublicKey, machine *string) *Manager {
	m := newManager(t, machine)
	m.Public = pub
	return m
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv := newPair(t)
	issued := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	key, err := Sign(priv, Claims{Customer: "Société Générale CI", Issued: issued})
	if err != nil {
		t.Fatal(err)
	}
	if !IsSigned(key) || key != NormalizeKey(key) {
		t.Fatalf("clé signée mal formée ou non normalisée : %s", key)
	}
	c, err := Verify(pub, key)
	if err != nil {
		t.Fatal(err)
	}
	if c.Customer != "Société Générale CI" || len(c.ID) != 16 || !c.Issued.Equal(issued) {
		t.Fatalf("contenu inattendu : %+v", c)
	}
	// Deux émissions donnent deux identifiants différents.
	key2, _ := Sign(priv, Claims{})
	c2, _ := Verify(pub, key2)
	if c2.ID == c.ID {
		t.Fatal("identifiants de licence identiques")
	}
	// Identifiant imposé (réémission d'une licence existante).
	key3, err := Sign(priv, Claims{ID: c.ID, Customer: "X"})
	if c3, err2 := Verify(pub, key3); err != nil || err2 != nil || c3.ID != c.ID {
		t.Fatalf("identifiant imposé : %v %v", err, err2)
	}
}

func TestVerifyRejectsTamperingAndOtherSigner(t *testing.T) {
	pub, priv := newPair(t)
	otherPub, otherPriv := newPair(t)
	key, _ := Sign(priv, Claims{Customer: "ACME"})

	if _, err := Verify(otherPub, key); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("clé publique d'un autre fournisseur : %v", err)
	}
	foreign, _ := Sign(otherPriv, Claims{Customer: "ACME"})
	if _, err := Verify(pub, foreign); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("licence signée par un autre : %v", err)
	}
	// Altération d'un caractère dans la charge utile, puis dans la signature.
	body := []byte(strings.TrimPrefix(key, signedPrefix))
	for _, i := range []int{3, len(body) - 3} {
		b := append([]byte(nil), body...)
		if b[i] == 'A' {
			b[i] = 'B'
		} else {
			b[i] = 'A'
		}
		if _, err := Verify(pub, signedPrefix+string(b)); err == nil {
			t.Fatalf("clé altérée en %d acceptée", i)
		}
	}
	for _, bad := range []string{"", "SGL1.", "SGL1.AAAA", "SGRD-ABCD-EFGH-JKLM-Y2DP", signedPrefix + "!!!!"} {
		if _, err := Verify(pub, bad); err == nil {
			t.Errorf("clé invalide acceptée : %q", bad)
		}
	}
	if _, err := Verify(nil, key); err == nil {
		t.Fatal("vérification sans clé publique acceptée")
	}
}

func TestSignRejectsBadInput(t *testing.T) {
	_, priv := newPair(t)
	if _, err := Sign(priv, Claims{Customer: strings.Repeat("x", 61)}); err == nil {
		t.Fatal("titulaire trop long accepté")
	}
	if _, err := Sign(priv, Claims{ID: "zz"}); err == nil {
		t.Fatal("identifiant invalide accepté")
	}
	if _, err := Sign(ed25519.PrivateKey("court"), Claims{}); err == nil {
		t.Fatal("clé privée invalide acceptée")
	}
}

func TestSignedModeActivation(t *testing.T) {
	pub, priv := newPair(t)
	_, otherPriv := newPair(t)
	mid := "poste-a"
	m := signedManager(t, pub, &mid)

	good, _ := Sign(priv, Claims{Customer: "ACME"})
	forged, _ := Sign(otherPriv, Claims{Customer: "ACME"})

	if _, err := m.Activate(forged); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("licence d'un autre fournisseur : %v", err)
	}
	// Les anciennes clés non signées sont refusées en mode signé.
	if _, err := m.Activate(MakeKey("ABCD", "EFGH", "JKLM")); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("clé non signée en mode signé : %v", err)
	}
	if s := m.Status(); s.State != StateNone || !s.SignedMode {
		t.Fatalf("rien ne devait être écrit : %+v", s)
	}

	// Clé collée sur plusieurs lignes, en minuscules : acceptée.
	pasted := strings.ToLower(good[:20] + "\n  " + good[20:])
	s, err := m.Activate(pasted)
	if err != nil || s.State != StateActive || !s.Signed || s.Customer != "ACME" {
		t.Fatalf("activation signée : %+v, %v", s, err)
	}
	if strings.Contains(s.MaskedKey, good[10:30]) || !strings.HasSuffix(s.MaskedKey, good[len(good)-6:]) || !strings.HasPrefix(s.MaskedKey, "SGL1.****") {
		t.Fatalf("masquage : %s", s.MaskedKey)
	}
	rec, _ := m.load()
	if rec.Customer != "ACME" || len(rec.LicenseID) != 16 {
		t.Fatalf("enregistrement : %+v", rec)
	}

	// Transfert : désactivation puis activation sur un autre poste.
	if _, err := m.Deactivate(); err != nil {
		t.Fatal(err)
	}
	mid = "poste-b"
	if s, err := m.Activate(good); err != nil || s.MachineID != "poste-b" {
		t.Fatalf("transfert : %+v, %v", s, err)
	}
}

func TestUnsignedModeRejectsSignedKey(t *testing.T) {
	_, priv := newPair(t)
	mid := "poste-a"
	m := newManager(t, &mid) // pas de clé publique : mode non signé
	key, _ := Sign(priv, Claims{})
	if _, err := m.Activate(key); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("clé signée en mode non signé : %v", err)
	}
}

func TestParsePublicKey(t *testing.T) {
	pub, _ := newPair(t)
	enc := base64.StdEncoding.EncodeToString(pub)
	if k, err := ParsePublicKey("  " + enc + "\r\n"); err != nil || !k.Equal(pub) {
		t.Fatalf("clé publique valide : %v", err)
	}
	if k, err := ParsePublicKey("  \n"); k != nil || err != nil {
		t.Fatalf("clé vide : %v %v", k, err)
	}
	for _, bad := range []string{"pas-du-base64", base64.StdEncoding.EncodeToString([]byte("court"))} {
		if _, err := ParsePublicKey(bad); err == nil {
			t.Errorf("clé publique invalide acceptée : %q", bad)
		}
	}
}

func TestEmbeddedKeyEmptyByDefault(t *testing.T) {
	if DefaultPublicKey() != nil {
		t.Skip("une clé publique de production est intégrée à ce binaire")
	}
	if m := NewManager(filepath.Join(t.TempDir(), "config.json")); m.Public != nil {
		t.Fatal("mode signé actif sans clé publique intégrée")
	}
}
