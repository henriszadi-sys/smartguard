package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// runCheck vérifie que le module peut démarrer et affiche un diagnostic lisible.
// Code de sortie : 0 = tout est bon, 1 = au moins un problème.
func runCheck(cfgPath string) int {
	bad := 0
	ok := func(msg string, a ...any) { fmt.Printf("[OK]      "+msg+"\n", a...) }
	ko := func(msg string, a ...any) { bad++; fmt.Printf("[PROBLÈME] "+msg+"\n", a...) }

	fmt.Printf("Diagnostic SmartGUARD v%s\nConfiguration : %s\n\n", version, cfgPath)
	if _, err := os.Stat(cfgPath); err != nil {
		ko("fichier de configuration introuvable : %v", err)
		return 1
	}
	st, err := NewStore(cfgPath)
	if err != nil {
		ko("configuration illisible : %v", err)
		return 1
	}
	c := st.Config()
	ok("configuration lue (module « %s », logiciel « %s »)", c.ModuleName, c.SoftwareName)
	if err := validate(c); err != nil {
		ko("configuration invalide : %v", err)
	} else {
		ok("dates : %s → %s, module %s", orDash(c.StartDate), orDash(c.EndDate), map[bool]string{true: "activé", false: "désactivé"}[c.Enabled])
	}
	if c.AdminPasswordHash == "" {
		ko("aucun mot de passe administrateur défini")
	} else {
		ok("mot de passe administrateur défini")
	}

	// Port d'écoute
	ln, err := net.Listen("tcp", c.Listen)
	if err != nil {
		// Déjà occupé : est-ce ce module qui tourne ?
		port := c.Listen[strings.LastIndex(c.Listen, ":")+1:]
		if isOurModule("http://127.0.0.1:" + port + c.AdminPath + "/api/status") {
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
		if reach, msg := testUpstream(c.Upstream); reach {
			ok("application %s : %s", c.Upstream, msg)
		} else {
			ko("application %s : %s", c.Upstream, msg)
		}
	} else {
		ok("mode « ligne de code » (pas de redirection vers l'application)")
	}

	// Droit d'écriture du journal
	logPath := strings.TrimSuffix(cfgPath, ".json") + ".log"
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

func isOurModule(url string) bool {
	cl := &http.Client{Timeout: 3 * time.Second}
	resp, err := cl.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var st Status
	return resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&st) == nil && st.ModuleName != ""
}
