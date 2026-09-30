package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardianos/service"
)

const version = "1.3.0"

type program struct {
	app    *App
	srv    *http.Server
	stopCh chan struct{}
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	c := p.app.store.Config()
	p.stopCh = make(chan struct{})
	go p.app.watch(p.stopCh)
	p.srv = &http.Server{Addr: c.Listen, Handler: p.app.Handler(), ReadHeaderTimeout: 15 * time.Second}
	p.app.logf("%s v%s démarré — écoute %s, application : %s, administration : %s/admin",
		c.ModuleName, version, c.Listen, orNone(c.Upstream), c.AdminPath)
	var err error
	if c.TLSCert != "" && c.TLSKey != "" {
		err = p.srv.ListenAndServeTLS(c.TLSCert, c.TLSKey)
	} else {
		err = p.srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		p.app.logf("ERREUR serveur : %v", err)
		os.Exit(1)
	}
}

func (p *program) Stop(s service.Service) error {
	if p.stopCh != nil {
		close(p.stopCh)
	}
	if p.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.srv.Shutdown(ctx)
	}
	p.app.logf("Module arrêté")
	return nil
}

func orNone(s string) string {
	if s == "" {
		return "(aucune — mode script seul)"
	}
	return s
}

func usage() {
	fmt.Fprintf(os.Stderr, `LicGuard v%s — module de rappel d'expiration de licence / contrat de support

Utilisation :
  licguard [options] <commande>

Commandes :
  (aucune)       ouvrir l'assistant d'installation graphique (double-clic)
  setup          idem ; sur un serveur sans écran : -setup-listen 0.0.0.0:8099
  run            lancer au premier plan (test)
  install        installer comme service (Windows / systemd)
  uninstall      désinstaller le service
  start | stop | restart   piloter le service
  status         afficher l'état du décompte
  check          diagnostic : configuration, port, application
  set-password   définir le mot de passe administrateur

Options :
  -config <fichier>   fichier de configuration (défaut : config.json à côté de l'exécutable)
  -name <nom>         nom du service système (défaut : celui de la configuration)
`, version)
}

func main() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	cfgPath := flag.String("config", filepath.Join(dir, "config.json"), "fichier de configuration")
	name := flag.String("name", "", "nom du service système")
	setupListen := flag.String("setup-listen", "", "adresse d'écoute de l'assistant (ex. 0.0.0.0:8099)")
	flag.Usage = usage
	flag.Parse()

	// Double-clic (aucune commande) ou « setup » : assistant d'installation graphique.
	if cmd := flag.Arg(0); cmd == "setup" || (cmd == "" && service.Interactive()) {
		if err := runSetup(*setupListen); err != nil {
			log.Println(err)
			pauseConsole()
			os.Exit(1)
		}
		return
	}

	abs, _ := filepath.Abs(*cfgPath)
	logPath := strings.TrimSuffix(abs, filepath.Ext(abs)) + ".log"
	cmd0 := flag.Arg(0)
	if cmd0 == "" || cmd0 == "run" {
		// Journal ouvert dès le lancement : toute erreur de démarrage y est consignée.
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
			log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
		log.SetFlags(log.Ldate | log.Ltime)
		log.Printf("Lancement v%s — %s %s (service Windows/systemd : %v)", version, exe, strings.Join(os.Args[1:], " "), !service.Interactive())
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ERREUR FATALE (panic) : %v", r)
				os.Exit(2)
			}
		}()
	}
	if cmd0 == "check" {
		os.Exit(runCheck(abs))
	}
	store, err := NewStore(abs)
	if err != nil {
		log.Fatalf("ERREUR configuration : %v", err)
	}
	c := store.Config()
	if *name != "" && *name != c.ServiceName {
		_ = store.UpdateConfig(func(x *Config) error { x.ServiceName = *name; return nil })
		c = store.Config()
	}
	app := &App{store: store, logPath: logPath, fails: map[string][]time.Time{}, sessions: map[string]time.Time{}}
	if err := app.setupProxy(c.Upstream); err != nil {
		log.Fatalf("ERREUR configuration : %v", err)
	}

	svcName := sanitizeName(c.ServiceName)
	svcCfg := &service.Config{
		Name:        svcName,
		DisplayName: c.ServiceName + " (rappel d'expiration)",
		Description: "Décompte d'expiration de licence / contrat de support pour " + c.SoftwareName,
		Arguments:   []string{"-config", abs, "run"},
	}
	prg := &program{app: app}
	s, err := service.New(prg, svcCfg)
	if err != nil {
		log.Fatalf("ERREUR service : %v", err)
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "", "run":
		if err := s.Run(); err != nil {
			log.Fatalf("ERREUR exécution du service « %s » : %v", svcName, err)
		}
	case "install", "uninstall", "start", "stop", "restart":
		if err := service.Control(s, cmd); err != nil {
			log.Fatalf("%s : %v (droits administrateur / root requis)", cmd, err)
		}
		fmt.Printf("Service « %s » : %s OK\n", svcName, cmd)
	case "status":
		st := computeStatus(c, store.State())
		fmt.Printf("Module      : %s (%s)\nLogiciel    : %s\nContrat     : %s → %s\nJours rest. : %d\nExpiré      : %v\nBandeau     : %v\nMessage     : %s\n",
			c.ModuleName, map[bool]string{true: "activé", false: "désactivé"}[c.Enabled],
			c.SoftwareName, c.StartDate, c.EndDate, st.DaysLeft, st.Expired, st.Show, st.Message)
	case "set-password":
		pw := os.Getenv("LICGUARD_PASSWORD")
		if pw == "" {
			fmt.Print("Nouveau mot de passe administrateur (8 caractères min.) : ")
			pw, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			pw = strings.TrimRight(pw, "\r\n")
		}
		if len(pw) < 8 {
			log.Fatal("mot de passe trop court")
		}
		h, err := hashPassword(pw)
		if err != nil {
			log.Fatal(err)
		}
		if err := store.UpdateConfig(func(x *Config) error { x.AdminPasswordHash = h; return nil }); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Mot de passe enregistré. Utilisateur : %s\n", c.AdminUser)
	default:
		usage()
		os.Exit(2)
	}
}

func sanitizeName(n string) string {
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
		return "LicGuard"
	}
	return b.String()
}
