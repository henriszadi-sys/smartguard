// Package wizard est l'assistant d'installation : il installe, modifie,
// renomme et désinstalle les modules d'un serveur.
package wizard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kardianos/service"

	"smartguard/internal/config"
	"smartguard/internal/license"
	"smartguard/internal/logging"
	"smartguard/internal/platform"
	"smartguard/internal/scheduler"
	"smartguard/internal/version"
	"smartguard/web"
)

// Installation : un module installé (un par logiciel contrôlé).
type Installation struct {
	Name   string `json:"name"`   // nom du module = nom du service
	Dir    string `json:"dir"`    // dossier d'installation
	Exe    string `json:"exe"`    // chemin de l'exécutable installé
	Config string `json:"config"` // chemin de config.json
	Port   int    `json:"port"`
}

var regMu sync.Mutex

func registryPath() string { return filepath.Join(platform.RegistryDir(), "installations.json") }

func loadRegistry() []Installation {
	var list []Installation
	b, err := os.ReadFile(registryPath())
	if errors.Is(err, os.ErrNotExist) {
		// Reprise des modules installés avant le renommage du produit.
		b, err = os.ReadFile(filepath.Join(platform.LegacyRegistryDir(), "installations.json"))
	}
	if err == nil {
		_ = json.Unmarshal(b, &list)
	}
	return list
}

func saveRegistry(list []Installation) error {
	if err := os.MkdirAll(platform.RegistryDir(), 0755); err != nil {
		return err
	}
	sort.Slice(list, func(i, j int) bool { return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name) })
	return config.WriteJSON(registryPath(), list)
}

func findInstall(name string) (Installation, bool) {
	for _, in := range loadRegistry() {
		if strings.EqualFold(in.Name, name) {
			return in, true
		}
	}
	return Installation{}, false
}

type nopProgram struct{}

func (nopProgram) Start(service.Service) error { return nil }
func (nopProgram) Stop(service.Service) error  { return nil }

func serviceConfig(name, exe, cfgPath, software string) *service.Config {
	return &service.Config{
		Name:        config.SanitizeName(name),
		DisplayName: name + " (rappel d'expiration)",
		Description: "Décompte d'expiration de licence / contrat de support — " + software,
		Executable:  exe,
		Arguments:   []string{"-config", cfgPath, "run"},
	}
}

func controlFor(name, exe, cfgPath, software string) (service.Service, error) {
	return service.New(nopProgram{}, serviceConfig(name, exe, cfgPath, software))
}

func svcStatusLabel(s service.Service) string {
	st, err := s.Status()
	if err != nil {
		return "non installé"
	}
	switch st {
	case service.StatusRunning:
		return "démarré"
	case service.StatusStopped:
		return "arrêté"
	}
	return "inconnu"
}

func stopAndWait(s service.Service) {
	_ = s.Stop()
	for i := 0; i < 30; i++ {
		if st, err := s.Status(); err != nil || st != service.StatusRunning {
			time.Sleep(500 * time.Millisecond) // laisse le processus se terminer
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// ---------------------------------------------------------------- serveur de l'assistant

type setupServer struct {
	token string
	quit  chan struct{}
	once  sync.Once
	mu    sync.Mutex // une seule installation à la fois
}

// Run ouvre l'assistant dans le navigateur (ou affiche son adresse sur un serveur sans écran).
func Run(listen string) error {
	if !platform.IsAdmin() {
		fmt.Println("Des droits administrateur sont nécessaires : confirmation demandée…")
		if err := platform.RelaunchElevated([]string{"setup"}); err != nil {
			fmt.Println("Impossible d'obtenir les droits administrateur :", err)
			platform.PauseConsole()
			return err
		}
		return nil
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	ss := &setupServer{token: hex.EncodeToString(b), quit: make(chan struct{})}

	remote := !platform.HasDesktop() && !strings.HasPrefix(listen, "127.0.0.1")
	if listen == "" {
		listen = "127.0.0.1:0"
		if remote {
			listen = "0.0.0.0:8099"
		}
	}
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir l'assistant sur %s : %w", listen, err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	srv := &http.Server{Handler: ss.routes(), ReadHeaderTimeout: 15 * time.Second}
	go func() { _ = srv.Serve(ln) }()

	local := fmt.Sprintf("http://127.0.0.1:%d/?t=%s", port, ss.token)
	fmt.Println("===============================================================")
	fmt.Println("  SmartGUARD — assistant d'installation")
	fmt.Println("===============================================================")
	if remote {
		fmt.Println("Aucun écran détecté. Ouvrez l'une de ces adresses depuis un navigateur :")
		for _, ip := range localIPs() {
			fmt.Printf("   http://%s:%d/?t=%s\n", ip, port, ss.token)
		}
		fmt.Printf("   (ou tunnel SSH : ssh -L %d:127.0.0.1:%d serveur  puis %s)\n", port, port, local)
	} else {
		fmt.Println("L'assistant s'ouvre dans votre navigateur. Si ce n'est pas le cas, copiez :")
		fmt.Println("   " + local)
		platform.OpenBrowser(local)
	}
	fmt.Println("\nNe fermez pas cette fenêtre avant d'avoir terminé.")

	select {
	case <-ss.quit:
	case <-time.After(2 * time.Hour):
	}
	time.Sleep(300 * time.Millisecond)
	_ = srv.Close()
	fmt.Println("Assistant fermé.")
	return nil
}

func localIPs() []string {
	var out []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLoopback() {
			out = append(out, ipn.IP.String())
		}
	}
	if len(out) == 0 {
		out = []string{"ADRESSE-DU-SERVEUR"}
	}
	return out
}

func (ss *setupServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.URL.Query().Get("t") != ss.token {
			http.Error(w, "Lien de l'assistant invalide : utilisez l'adresse affichée dans la fenêtre d'installation.", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		_, _ = w.Write(web.SetupHTML)
	})
	api := func(path string, h func(w http.ResponseWriter, r *http.Request)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Setup-Token") != ss.token {
				http.Error(w, "accès refusé", http.StatusForbidden)
				return
			}
			h(w, r)
		})
	}
	api("/api/info", ss.apiInfo)
	api("/api/services", func(w http.ResponseWriter, r *http.Request) { respondJSON(w, platform.ListSystemServices()) })
	api("/api/existing", ss.apiExisting)
	api("/api/check", ss.apiCheck)
	api("/api/install", ss.apiInstall)
	api("/api/uninstall", ss.apiUninstall)
	api("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]bool{"ok": true})
		ss.once.Do(func() { close(ss.quit) })
	})
	return mux
}

type installView struct {
	Installation
	SoftwareName string `json:"software_name"`
	EndDate      string `json:"end_date"`
	Enabled      bool   `json:"enabled"`
	DaysLeft     int    `json:"days_left"`
	Level        string `json:"level"`
	Deadlines    int    `json:"deadlines"` // nombre d'échéances suivies
	Status       string `json:"status"`
	AdminURL     string `json:"admin_url"`
}

func (ss *setupServer) apiInfo(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	var views []installView
	for _, in := range loadRegistry() {
		v := installView{Installation: in, Status: "non installé"}
		if st, err := config.OpenExisting(in.Config); err == nil {
			c := st.Config()
			s := scheduler.ComputeStatus(c, st.State(), time.Now())
			v.SoftwareName, v.EndDate, v.Enabled, v.DaysLeft, v.Level = c.SoftwareName, s.EndDate, c.Enabled, s.DaysLeft, s.Level
			v.Deadlines = len(c.Deadlines)
			v.AdminURL = fmt.Sprintf("http://%s:%d%s/admin", strings.ToLower(host), in.Port, c.AdminPath)
			if svc, err := controlFor(in.Name, in.Exe, in.Config, c.SoftwareName); err == nil {
				v.Status = svcStatusLabel(svc)
			}
		}
		views = append(views, v)
	}
	if views == nil {
		views = []installView{}
	}
	respondJSON(w, map[string]any{
		"version":      version.Number,
		"os":           runtime.GOOS,
		"hostname":     host,
		"installs":     views,
		"dir_template": platform.DefaultInstallDir("{NOM}"),
		"messages":     defaultMessages(),
		"license":      licenseManager().Status(),
	})
}

// defaultMessages : messages par défaut (rappel, échéance passée) par type d'échéance.
func defaultMessages() map[string][2]string {
	m := map[string][2]string{}
	for _, k := range []string{config.KindContrat, config.KindLicence, config.KindAbonnement} {
		a, b := config.DefaultMessages(k)
		m[k] = [2]string{a, b}
	}
	return m
}

func (ss *setupServer) apiExisting(w http.ResponseWriter, r *http.Request) {
	in, ok := findInstall(r.URL.Query().Get("name"))
	if !ok {
		http.Error(w, "installation introuvable", http.StatusNotFound)
		return
	}
	st, err := config.OpenExisting(in.Config)
	if err != nil {
		http.Error(w, "configuration illisible : "+err.Error(), http.StatusInternalServerError)
		return
	}
	c := st.Config()
	c.AdminPasswordHash = ""
	respondJSON(w, map[string]any{"install": in, "config": c})
}

type checkReq struct {
	Existing string `json:"existing"`
	Port     int    `json:"port"`
	Upstream string `json:"upstream"`
}

func portFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func (ss *setupServer) apiCheck(w http.ResponseWriter, r *http.Request) {
	var q checkReq
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&q)
	res := map[string]any{}
	if q.Port > 0 {
		ok := portFree(q.Port)
		if !ok {
			if in, found := findInstall(q.Existing); found && in.Port == q.Port {
				ok = true // port déjà utilisé par ce même module
			}
		}
		res["port_ok"] = ok
	}
	if q.Upstream != "" {
		res["upstream_ok"], res["upstream_msg"] = TestUpstream(q.Upstream)
	}
	respondJSON(w, res)
}

// TestUpstream vérifie que l'application surveillée répond à l'adresse indiquée.
func TestUpstream(u string) (bool, string) {
	pu, err := url.Parse(u)
	if err != nil || pu.Host == "" || (pu.Scheme != "http" && pu.Scheme != "https") {
		return false, "adresse invalide (ex. http://127.0.0.1:8081)"
	}
	cl := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := cl.Get(u)
	if err != nil {
		return false, "application injoignable à cette adresse"
	}
	resp.Body.Close()
	return true, fmt.Sprintf("application joignable (réponse %d)", resp.StatusCode)
}

type installReq struct {
	Existing        string   `json:"existing"`
	ModuleName      string   `json:"module_name"`
	SoftwareName    string   `json:"software_name"`
	SupplierContact string   `json:"supplier_contact"`
	Enabled         bool     `json:"enabled"`
	Kind            string   `json:"kind"`  // type de la première échéance
	Label           string   `json:"label"` // libellé de la première échéance
	StartDate       string   `json:"start_date"`
	EndDate         string   `json:"end_date"`
	StopOnEnd       bool     `json:"stop_on_end"`
	WarningDays     int      `json:"warning_days"`
	Message         string   `json:"message"`
	ExpiredMessage  string   `json:"expired_message"`
	Services        []string `json:"services"`
	BlockedURLs     []string `json:"blocked_urls"`
	Scripts         []string `json:"scripts"`
	Mode            string   `json:"mode"` // proxy | script
	Port            int      `json:"port"`
	Upstream        string   `json:"upstream"`
	InstallDir      string   `json:"install_dir"`
	Password        string   `json:"password"`
	Shortcut        bool     `json:"shortcut"`
	LicenseKey      string   `json:"license_key"` // clé de licence SmartGUARD du serveur (saisie ou fichier)
}

// RequireLicenseOnUpdate : la mise à jour d'un module déjà installé exige-t-elle
// aussi une licence active ? Non par défaut (point à valider) : un module installé
// avant la licence obligatoire peut être mis à jour, la licence restant signalée
// comme à activer.
const RequireLicenseOnUpdate = false

// licenseManager : licence du serveur (remplaçable dans les tests).
var licenseManager = func() *license.Manager { return license.NewManager("") }

// checkLicense applique la règle « sans licence valide, le module ne s'installe pas ».
// Renvoie le libellé de l'étape pour le journal d'installation.
func checkLicense(lm *license.Manager, key string, existing bool) (string, error) {
	st, err := lm.RequireForInstall(key)
	switch {
	case err == nil:
		return "licence " + st.MaskedKey + " active sur ce serveur", nil
	case existing && !RequireLicenseOnUpdate && errors.Is(err, license.ErrLicenseRequired):
		return "mise à jour sans licence SmartGUARD : licence à activer depuis l'administration", nil
	}
	return "", err
}

type step struct {
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func (ss *setupServer) apiInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var q installReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&q); err != nil {
		writeErr(w, "données invalides")
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	res, err := doInstall(q)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "steps": res["steps"]})
		return
	}
	respondJSON(w, res)
}

func writeErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func doInstall(q installReq) (map[string]any, error) {
	steps := []step{}
	res := map[string]any{}
	logDir := ""
	defer func() {
		if logDir == "" {
			return
		}
		var b strings.Builder
		fmt.Fprintf(&b, "=== Installation %s — SmartGUARD v%s ===\r\n", time.Now().Format("02/01/2006 15:04:05"), version.Number)
		for _, st := range steps {
			mark := "OK "
			if !st.OK {
				mark = "ÉCHEC"
			}
			fmt.Fprintf(&b, "[%s] %s\r\n", mark, st.Label)
			if st.Detail != "" {
				fmt.Fprintf(&b, "      %s\r\n", strings.ReplaceAll(st.Detail, "\n", "\r\n      "))
			}
		}
		b.WriteString("\r\n")
		logging.AppendInstall(logDir, b.String())
	}()
	add := func(label string, err error, detail string) error {
		s := step{Label: label, OK: err == nil, Detail: detail}
		if err != nil {
			s.Detail = err.Error()
		}
		steps = append(steps, s)
		res["steps"] = steps
		return err
	}

	// --- contrôles
	q.ModuleName = strings.TrimSpace(q.ModuleName)
	name := config.SanitizeName(q.ModuleName)
	if q.ModuleName == "" || name == "" {
		return res, errors.New("indiquez le nom du module")
	}
	if strings.TrimSpace(q.SoftwareName) == "" {
		return res, errors.New("indiquez le nom du logiciel")
	}
	if q.Port < 1 || q.Port > 65535 {
		return res, errors.New("port d'écoute invalide")
	}
	var old Installation
	existing := false
	if q.Existing != "" {
		old, existing = findInstall(q.Existing)
		if !existing {
			return res, errors.New("installation à modifier introuvable")
		}
	}
	if other, found := findInstall(name); found && (!existing || !strings.EqualFold(other.Name, old.Name)) {
		return res, fmt.Errorf("un module nommé « %s » est déjà installé", other.Name)
	}
	if !existing && len(q.Password) < 8 {
		return res, errors.New("le mot de passe administrateur doit contenir au moins 8 caractères")
	}
	if q.Password != "" && len(q.Password) < 8 {
		return res, errors.New("le mot de passe administrateur doit contenir au moins 8 caractères")
	}
	if !(existing && old.Port == q.Port) && !portFree(q.Port) {
		return res, fmt.Errorf("le port %d est déjà utilisé sur ce serveur : choisissez-en un autre", q.Port)
	}
	upstream := ""
	if q.Mode == "proxy" {
		pu, err := url.Parse(strings.TrimSpace(q.Upstream))
		if err != nil || pu.Host == "" {
			return res, errors.New("adresse de l'application invalide (ex. http://127.0.0.1:8081)")
		}
		upstream = strings.TrimRight(pu.String(), "/")
	}
	first := config.Deadline{Kind: q.Kind, Label: q.Label, StartDate: q.StartDate, EndDate: q.EndDate, StopOnEnd: q.StopOnEnd,
		WarningDays: q.WarningDays, Message: q.Message, ExpiredMessage: q.ExpiredMessage,
		Services: q.Services, BlockedURLs: q.BlockedURLs, Scripts: q.Scripts}
	if err := config.ValidateDeadline(first); err != nil {
		return res, err
	}
	// Licence SmartGUARD du serveur : contrôlée avant toute copie de fichier.
	licDetail, licErr := checkLicense(licenseManager(), q.LicenseKey, existing)
	if err := add("Licence SmartGUARD", licErr, licDetail); err != nil {
		return res, err
	}
	dir := strings.TrimSpace(q.InstallDir)
	if existing {
		dir = old.Dir
	}
	if dir == "" {
		dir = platform.DefaultInstallDir(name)
	}
	if !filepath.IsAbs(dir) {
		return res, errors.New("le dossier d'installation doit être un chemin complet")
	}
	exeDst := filepath.Join(dir, platform.ExeFileName(name))
	cfgPath := filepath.Join(dir, "config.json")
	renamed := existing && !strings.EqualFold(old.Name, name)

	// --- arrêt de l'ancienne version
	if existing {
		if svc, err := controlFor(old.Name, old.Exe, old.Config, q.SoftwareName); err == nil {
			stopAndWait(svc)
			if renamed {
				_ = svc.Uninstall()
				platform.CloseFirewall(old.Name)
				platform.RemoveShortcut(old.Name)
			}
		}
		_ = add("Arrêt du module existant", nil, old.Name)
	}

	// --- copie des fichiers
	if err := add("Création du dossier", os.MkdirAll(dir, 0755), dir); err != nil {
		return res, err
	}
	logDir = dir
	self, _ := os.Executable()
	if !samePath(self, exeDst) {
		if err := add("Copie du programme", copyFileRetry(self, exeDst), exeDst); err != nil {
			return res, err
		}
	} else {
		_ = add("Programme déjà en place", nil, exeDst)
	}
	if renamed && !samePath(old.Exe, exeDst) {
		_ = os.Remove(old.Exe)
	}

	// --- configuration
	st, err := config.NewStore(cfgPath)
	if err == nil {
		err = st.UpdateConfig(func(c *config.Config) error {
			c.ModuleName, c.ServiceName = q.ModuleName, q.ModuleName
			c.SoftwareName = strings.TrimSpace(q.SoftwareName)
			c.SupplierContact = strings.TrimSpace(q.SupplierContact)
			c.Enabled = q.Enabled
			// L'assistant règle la première échéance ; les autres sont conservées.
			if len(c.Deadlines) > 0 {
				first.ID = c.Deadlines[0].ID
				c.Deadlines[0] = first
			} else {
				c.Deadlines = []config.Deadline{first}
			}
			c.Listen = fmt.Sprintf(":%d", q.Port)
			c.Upstream = upstream
			if q.Password != "" {
				h, err := config.HashPassword(q.Password)
				if err != nil {
					return err
				}
				c.AdminPasswordHash = h
			}
			return nil
		})
	}
	if err := add("Enregistrement de la configuration", err, cfgPath); err != nil {
		return res, err
	}

	adminPath := st.Config().AdminPath // chemin conservé lors d'une mise à jour

	// --- service système
	svc, err := controlFor(name, exeDst, cfgPath, q.SoftwareName)
	if err != nil {
		return res, add("Préparation du service", err, "")
	}
	if !existing || renamed {
		err = svc.Install()
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "exist") {
			_ = svc.Uninstall() // reste d'une ancienne installation
			err = svc.Install()
		}
		if err := add("Installation du service « "+config.SanitizeName(name)+" »", err, ""); err != nil {
			return res, err
		}
	}
	_ = add("Ouverture du port dans le pare-feu", nil, platform.OpenFirewall(name, q.Port))
	if err := svc.Start(); err != nil {
		_ = add("Démarrage du service", err, "")
		_ = add("Diagnostic", nil, diagnose(name, exeDst, cfgPath))
		return res, fmt.Errorf("le service n'a pas pu démarrer : %v — voir le diagnostic ci-dessous", err)
	}
	_ = add("Démarrage du service", nil, "")
	ok := false
	for i := 0; i < 30 && !ok; i++ {
		time.Sleep(500 * time.Millisecond)
		ok = IsOurModule(fmt.Sprintf("http://127.0.0.1:%d%s/api/status", q.Port, adminPath))
	}
	if !ok {
		_ = add("Vérification du fonctionnement", errors.New("le module ne répond pas"), "")
		_ = add("Diagnostic", nil, diagnose(name, exeDst, cfgPath))
	} else {
		_ = add("Vérification du fonctionnement", nil, "le module répond")
	}

	// --- raccourci + registre
	host, _ := os.Hostname()
	adminURL := fmt.Sprintf("http://%s:%d%s/admin", strings.ToLower(host), q.Port, adminPath)
	if q.Shortcut && runtime.GOOS == "windows" {
		_ = add("Raccourci sur le bureau", platform.CreateShortcut(q.ModuleName, adminURL), "")
	}
	regMu.Lock()
	list := []Installation{}
	for _, in := range loadRegistry() {
		if !strings.EqualFold(in.Name, old.Name) && !strings.EqualFold(in.Name, name) {
			list = append(list, in)
		}
	}
	list = append(list, Installation{Name: name, Dir: dir, Exe: exeDst, Config: cfgPath, Port: q.Port})
	_ = saveRegistry(list)
	regMu.Unlock()

	res["admin_url"] = adminURL
	res["snippet"] = fmt.Sprintf(`<script src="http://%s:%d%s/banner.js" defer></script>`, strings.ToLower(host), q.Port, adminPath)
	res["mode"] = q.Mode
	res["name"] = name
	return res, nil
}

func (ss *setupServer) apiUninstall(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Name        string `json:"name"`
		DeleteFiles bool   `json:"delete_files"`
	}
	if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&q) != nil {
		writeErr(w, "requête invalide")
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	in, ok := findInstall(q.Name)
	if !ok {
		writeErr(w, "installation introuvable")
		return
	}
	steps := []step{}
	if svc, err := controlFor(in.Name, in.Exe, in.Config, ""); err == nil {
		stopAndWait(svc)
		err := svc.Uninstall()
		steps = append(steps, step{Label: "Suppression du service", OK: err == nil, Detail: errStr(err)})
	}
	platform.CloseFirewall(in.Name)
	platform.RemoveShortcut(in.Name)
	if q.DeleteFiles {
		self, _ := os.Executable()
		var err error
		if samePath(filepath.Dir(self), in.Dir) {
			err = errors.New("l'assistant tourne depuis ce dossier : supprimez-le manuellement après fermeture")
		} else {
			err = os.RemoveAll(in.Dir)
		}
		steps = append(steps, step{Label: "Suppression des fichiers", OK: err == nil, Detail: errStr(err)})
	} else {
		steps = append(steps, step{Label: "Fichiers conservés", OK: true, Detail: in.Dir})
	}
	regMu.Lock()
	list := []Installation{}
	for _, x := range loadRegistry() {
		if !strings.EqualFold(x.Name, in.Name) {
			list = append(list, x)
		}
	}
	_ = saveRegistry(list)
	regMu.Unlock()
	respondJSON(w, map[string]any{"ok": true, "steps": steps})
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func samePath(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func copyFileRetry(src, dst string) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = copyFile(src, dst); err == nil {
			return nil
		}
		time.Sleep(time.Second) // le fichier peut encore être verrouillé par le service qui s'arrête
	}
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}

// diagnose rassemble les informations utiles quand le service ne démarre pas.
func diagnose(name, exe, cfgPath string) string {
	var b strings.Builder
	if _, err := os.Stat(exe); err != nil {
		b.WriteString("• Le programme installé a disparu (" + exe + ").\n  Cause probable : l'antivirus l'a mis en quarantaine. Ajoutez une exclusion pour ce dossier puis relancez l'installation.\n")
		return b.String()
	}
	out, _ := exec.Command(exe, "-config", cfgPath, "check").CombinedOutput()
	b.WriteString("• Vérification du module :\n" + strings.TrimSpace(string(out)) + "\n")
	logPath := logging.PathFor(cfgPath)
	if data, err := os.ReadFile(logPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		b.WriteString("• Dernières lignes du journal (" + logPath + ") :\n" + strings.Join(lines, "\n") + "\n")
	} else {
		b.WriteString("• Aucun journal écrit : le service n'a pas lancé le programme (voir l'état du service ci-dessous).\n")
	}
	b.WriteString(platform.ServiceDiagnostics(config.SanitizeName(name)))
	return b.String()
}

// IsOurModule indique si l'adresse répond comme l'API d'état d'un module SmartGUARD.
func IsOurModule(url string) bool {
	cl := &http.Client{Timeout: 3 * time.Second}
	resp, err := cl.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var st scheduler.Status
	return resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&st) == nil && st.ModuleName != ""
}

func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
