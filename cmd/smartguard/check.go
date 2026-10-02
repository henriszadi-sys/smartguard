package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"smartguard/internal/config"
	"smartguard/internal/logging"
	"smartguard/internal/platform"
	"smartguard/internal/version"
	"smartguard/internal/wizard"
)

// runCheck vérifie que le module peut démarrer et affiche un diagnostic lisible.
// Code de sortie : 0 = tout est bon, 1 = au moins un problème.
func runCheck(cfgPath string) int {
	bad := 0
	ok := func(msg string, a ...any) { fmt.Printf("[OK]      "+msg+"\n", a...) }
	ko := func(msg string, a ...any) { bad++; fmt.Printf("[PROBLÈME] "+msg+"\n", a...) }

	fmt.Printf("Diagnostic SmartGUARD v%s\nConfiguration : %s\n\n", version.Number, cfgPath)
	if sysOK, label := platform.CheckSystem(); sysOK {
		ok("système : %s", label)
	} else {
		ko("système : %s", label)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		ko("fichier de configuration introuvable : %v", err)
		return 1
	}
	st, err := config.NewStore(cfgPath)
	if err != nil {
		ko("configuration illisible : %v", err)
		return 1
	}
	c := st.Config()
	ok("configuration lue (module « %s », logiciel « %s »)", c.ModuleName, c.SoftwareName)
	if err := config.Validate(c); err != nil {
		ko("configuration invalide : %v", err)
	} else {
		ok("%d échéance(s), module %s", len(c.Deadlines), map[bool]string{true: "activé", false: "désactivé"}[c.Enabled])
		for _, d := range c.Deadlines {
			ok("  %s « %s » : %s → %s, arrêt à la fin : %v", d.ID, d.Name(), orDash(d.StartDate), d.EndDate, d.StopOnEnd)
		}
	}
	if c.AdminPasswordHash == "" {
		ko("aucun mot de passe administrateur défini")
	} else {
		ok("mot de passe du technicien défini")
	}
	if c.ClientPasswordHash == "" {
		ok("accès client désactivé (le technicien peut le créer)")
	} else {
		ok("accès client « %s » actif (consultation et réactivation)", c.ClientUser)
	}

	// Port d'écoute
	ln, err := net.Listen("tcp", c.Listen)
	if err != nil {
		// Déjà occupé : est-ce ce module qui tourne ?
		port := c.Listen[strings.LastIndex(c.Listen, ":")+1:]
		if wizard.IsOurModule(wizard.Scheme(c.TLSCert, c.TLSKey) + "://127.0.0.1:" + port + c.AdminPath + "/api/status") {
			ok("le port %s est utilisé par ce module (déjà démarré)", c.Listen)
		} else {
			ko("impossible d'écouter sur %s : %v — port occupé par un autre programme ou bloqué", c.Listen, err)
		}
	} else {
		ln.Close()
		ok("port %s disponible", c.Listen)
	}

	// Application surveillée (mode automatique)
	if c.Upstream != "" {
		if reach, msg := wizard.TestUpstream(c.Upstream); reach {
			ok("application %s : %s", c.Upstream, msg)
		} else {
			ko("application %s : %s", c.Upstream, msg)
		}
	} else {
		ok("mode « ligne de code » (pas de redirection vers l'application)")
	}

	// Droit d'écriture du journal
	logPath := logging.PathFor(cfgPath)
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err != nil {
		ko("impossible d'écrire le journal %s : %v", logPath, err)
	} else {
		f.Close()
		ok("journal accessible : %s", logPath)
	}

	if bad == 0 {
		fmt.Println("\nAucun problème détecté.")
		return 0
	}
	fmt.Printf("\n%d problème(s) détecté(s).\n", bad)
	return 1
}

func orDash(s string) string {
	if s == "" {
		return "–"
	}
	return s
}
