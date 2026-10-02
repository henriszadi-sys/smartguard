// Commande smartguard : assistant d'installation, service du module et
// commandes d'exploitation (status, check, set-password).
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

	"smartguard/internal/actions"
	"smartguard/internal/admin"
	"smartguard/internal/config"
	"smartguard/internal/license"
	"smartguard/internal/logging"
	"smartguard/internal/platform"
	"smartguard/internal/proxy"
	"smartguard/internal/scheduler"
	"smartguard/internal/version"
	"smartguard/internal/wizard"
)

type program struct {
	store   *config.Store
	lic     *license.Manager
	log     *logging.Logger
	admin   *admin.Server
	watcher *scheduler.Watcher
	srv     *http.Server
	stopCh  chan struct{}
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	c := p.store.Config()
	p.stopCh = make(chan struct{})
	go p.watcher.Run(p.stopCh)
	p.srv = &http.Server{Addr: c.Listen, Handler: p.admin.Handler(), ReadHeaderTimeout: 15 * time.Second}
	p.log.Printf("%s v%s démarré — écoute %s, application : %s, administration : %s/admin",
		c.ModuleName, version.Number, c.Listen, orNone(c.Upstream), c.AdminPath)
	if ls := p.lic.Status(); ls.State != license.StateNone {
		p.log.Printf("Licence %s : %s", ls.MaskedKey, ls.Message)
	}
	var err error
	if c.TLSCert != "" && c.TLSKey != "" {
		err = p.srv.ListenAndServeTLS(c.TLSCert, c.TLSKey)
	} else {
		err = p.srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		p.log.Printf("ERREUR serveur : %v", err)
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
	p.log.Printf("Module arrêté")
	return nil
}

// runLicense : commande « license » — état (défaut), activation ou désactivation.
// La clé complète n'est jamais affichée ni journalisée.
func runLicense(lic *license.Manager, logger *logging.Logger, args []string) {
	printStatus := func(s license.Status) {
		fmt.Printf("Serveur     : %s\nLicence     : %s\nActivée le  : %s\nÉtat        : %s — %s\n",
			s.MachineID, orDash(s.MaskedKey), orDash(s.ActivatedAt), s.State, s.Message)
	}
	switch {
	case len(args) == 0:
		printStatus(lic.Status())
	case args[0] == "activate" && len(args) == 2:
		s, err := lic.Activate(args[1])
		if err != nil {
			log.Fatal(err)
		}
		logger.Printf("Licence %s activée sur le poste %s (ligne de commande)", s.MaskedKey, s.MachineID)
		printStatus(s)
	case args[0] == "deactivate" && len(args) == 1:
		rec, err := lic.Deactivate()
		if err != nil {
			log.Fatal(err)
		}
		logger.Printf("Licence %s désactivée (poste %s, ligne de commande)", license.Mask(rec.Key), rec.MachineID)
		fmt.Printf("Licence %s désactivée. Elle peut être activée sur un autre poste.\n", license.Mask(rec.Key))
	default:
		usage()
		os.Exit(2)
	}
}

func orNone(s string) string {
	if s == "" {
		return "(aucune — mode script seul)"
	}
	return s
}

func usage() {
	fmt.Fprintf(os.Stderr, `SmartGUARD v%s — module de rappel d'expiration de licence / contrat de support

Utilisation :
  smartguard [options] <commande>

Commandes :
  (aucune)       ouvrir l'assistant d'installation graphique (double-clic)
  setup          idem ; sur un serveur sans écran : -setup-listen 0.0.0.0:8099
  run            lancer au premier plan (test)
  install        installer comme service (Windows / systemd)
  uninstall      désinstaller le service
  start | stop | restart   piloter le service
  status         afficher l'état du décompte
  check          diagnostic : configuration, port, application
  license [activate <clé> | deactivate]   licence SmartGUARD du serveur (exigée pour installer)
  set-password   définir le mot de passe administrateur

Options :
  -config <fichier>   fichier de configuration (défaut : config.json à côté de l'exécutable)
  -name <nom>         nom du service système (défaut : celui de la configuration)
`, version.Number)
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
		if err := wizard.Run(*setupListen); err != nil {
			log.Println(err)
			platform.PauseConsole()
			os.Exit(1)
		}
		return
	}

	abs, _ := filepath.Abs(*cfgPath)
	logPath := logging.PathFor(abs)
	cmd0 := flag.Arg(0)
	if cmd0 == "" || cmd0 == "run" {
		// Journal ouvert dès le lancement : toute erreur de démarrage y est consignée.
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
			log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
		log.SetFlags(log.Ldate | log.Ltime)
		log.Printf("Lancement v%s — %s %s (service Windows/systemd : %v)", version.Number, exe, strings.Join(os.Args[1:], " "), !service.Interactive())
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
	store, err := config.NewStore(abs)
	if err != nil {
		log.Fatalf("ERREUR configuration : %v", err)
	}
	c := store.Config()
	if *name != "" && *name != c.ServiceName {
		_ = store.UpdateConfig(func(x *config.Config) error { x.ServiceName = *name; return nil })
		c = store.Config()
	}
	logger := logging.New(logPath)
	px, err := proxy.New(c.Upstream, store, logger.Printf)
	if err != nil {
		log.Fatalf("ERREUR configuration : %v", err)
	}
	watcher := &scheduler.Watcher{
		Store:  store,
		Expire: func(c config.Config, d config.Deadline) { actions.Enforce(c, d, logger.Printf) },
	}
	lic := license.NewManager(abs)
	srv := &admin.Server{
		Store:   store,
		Log:     logger,
		License: lic,
		Proxy:   px,
		Restore: func(ds []config.Deadline) { actions.Restore(ds, logger.Printf) },
		Changed: watcher.Tick,
	}

	svcName := config.SanitizeName(c.ServiceName)
	svcCfg := &service.Config{
		Name:        svcName,
		DisplayName: c.ServiceName + " (rappel d'expiration)",
		Description: "Décompte d'expiration de licence / contrat de support pour " + c.SoftwareName,
		Arguments:   []string{"-config", abs, "run"},
	}
	prg := &program{store: store, lic: lic, log: logger, admin: srv, watcher: watcher}
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
	case "install":
		// Sans licence valide, le module ne s'installe pas (cahier des charges, section 4).
		if ls := lic.Status(); ls.State != license.StateActive {
			log.Fatalf("installation refusée : %s\nActivez d'abord la licence du serveur : smartguard license activate <clé>", ls.Message)
		}
		if err := service.Control(s, cmd); err != nil {
			log.Fatalf("%s : %v (droits administrateur / root requis)", cmd, err)
		}
		fmt.Printf("Service « %s » : %s OK\n", svcName, cmd)
	case "uninstall", "start", "stop", "restart":
		if err := service.Control(s, cmd); err != nil {
			log.Fatalf("%s : %v (droits administrateur / root requis)", cmd, err)
		}
		fmt.Printf("Service « %s » : %s OK\n", svcName, cmd)
	case "status":
		st := scheduler.ComputeStatus(c, store.State(), time.Now())
		fmt.Printf("Module      : %s (%s)\nLogiciel    : %s\nÉchéances   : %d\nBandeau     : %v\nMessage     : %s\n",
			c.ModuleName, map[bool]string{true: "activé", false: "désactivé"}[c.Enabled],
			c.SoftwareName, len(st.Deadlines), st.Show, st.Message)
		for _, d := range st.Deadlines {
			name := d.Label
			if d.Label != d.KindLabel {
				name += " (" + d.KindLabel + ")"
			}
			fmt.Printf("\n[%s] %s\n  Période     : %s → %s\n  Jours rest. : %d\n  Expirée     : %v\n  Arrêt à la fin : %v (arrêtée : %v)\n",
				d.ID, name, orDash(d.StartDate), orDash(d.EndDate), d.DaysLeft, d.Expired, d.StopOnEnd, d.Stopped)
			if d.ActionsDoneAt != "" {
				fmt.Printf("  Actions exécutées le %s\n", d.ActionsDoneAt)
			}
		}
	case "license":
		runLicense(lic, logger, flag.Args()[1:])
	case "set-password":
		pw := os.Getenv("SMARTGUARD_PASSWORD")
		if pw == "" {
			fmt.Print("Nouveau mot de passe administrateur (8 caractères min.) : ")
			pw, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			pw = strings.TrimRight(pw, "\r\n")
		}
		if len(pw) < 8 {
			log.Fatal("mot de passe trop court")
		}
		h, err := config.HashPassword(pw)
		if err != nil {
			log.Fatal(err)
		}
		if err := store.UpdateConfig(func(x *config.Config) error { x.AdminPasswordHash = h; return nil }); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Mot de passe enregistré. Utilisateur : %s\n", c.AdminUser)
	default:
		usage()
		os.Exit(2)
	}
}
