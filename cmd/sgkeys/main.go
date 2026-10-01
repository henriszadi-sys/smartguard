// Commande sgkeys : outil du fournisseur pour émettre les licences signées de
// SmartGUARD. Il n'est pas livré aux clients : il détient la clé privée.
//
//	sgkeys keygen -out fournisseur.sgpriv      crée la paire de clés
//	sgkeys issue  -key fournisseur.sgpriv -customer "Société X"   émet une licence
//	sgkeys verify -pub <clé publique ou fichier> <licence>        contrôle une licence
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"smartguard/internal/license"
)

func usage() {
	fmt.Fprint(os.Stderr, `sgkeys — émission des licences signées SmartGUARD

Commandes :
  keygen -out <fichier>                 crée la clé privée (à garder secrète, hors du dépôt)
                                        et affiche la clé publique à intégrer au module
  issue  -key <fichier> [-customer <titulaire>] [-id <16 hex>]
                                        émet une licence signée et l'affiche
  verify -pub <clé publique | fichier> <licence>
                                        vérifie une licence et affiche son contenu
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "issue":
		err = issue(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "", "fichier de la clé privée à créer")
	_ = fs.Parse(args)
	if *out == "" {
		return errors.New("indiquez -out <fichier>")
	}
	pub, priv, err := license.GenerateKeyPair()
	if err != nil {
		return err
	}
	// O_EXCL : ne jamais écraser une clé privée existante.
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("création de %s : %w", *out, err)
	}
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(priv) + "\n"); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("Clé privée écrite dans %s : gardez-la secrète, sauvegardez-la, ne la versionnez jamais.\n\n", *out)
	fmt.Println("Clé publique (à copier dans internal/license/public.key, puis reconstruire les exécutables) :")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	return nil
}

func readPrivate(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("fichier de clé privée invalide")
	}
	return ed25519.PrivateKey(raw), nil
}

func issue(args []string) error {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	keyFile := fs.String("key", "", "fichier de la clé privée")
	customer := fs.String("customer", "", "titulaire de la licence (optionnel, 60 octets max)")
	id := fs.String("id", "", "identifiant imposé (16 caractères hexadécimaux) ; aléatoire par défaut")
	_ = fs.Parse(args)
	if *keyFile == "" {
		return errors.New("indiquez -key <fichier>")
	}
	priv, err := readPrivate(*keyFile)
	if err != nil {
		return err
	}
	key, err := license.Sign(priv, license.Claims{ID: *id, Customer: *customer})
	if err != nil {
		return err
	}
	fmt.Println(key)
	return nil
}

func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pubArg := fs.String("pub", "", "clé publique (base64) ou fichier qui la contient")
	_ = fs.Parse(args)
	if *pubArg == "" || fs.NArg() != 1 {
		return errors.New("utilisation : verify -pub <clé publique | fichier> <licence>")
	}
	src := *pubArg
	if b, err := os.ReadFile(src); err == nil {
		src = string(b)
	}
	pub, err := license.ParsePublicKey(src)
	if err != nil || pub == nil {
		return errors.New("clé publique invalide ou vide")
	}
	c, err := license.Verify(pub, license.NormalizeKey(fs.Arg(0)))
	if err != nil {
		return err
	}
	fmt.Printf("Licence valide\nIdentifiant : %s\nTitulaire   : %s\nÉmise le    : %s\n", c.ID, orDash(c.Customer), c.Issued.Format("02/01/2006 15:04"))
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "–"
	}
	return s
}
