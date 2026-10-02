// Package admin sert les routes du module : bandeau, état public, page de
// connexion et interface web d'administration.
package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"smartguard/internal/config"
	"smartguard/internal/license"
	"smartguard/internal/logging"
	"smartguard/internal/proxy"
	"smartguard/internal/scheduler"
	"smartguard/web"
)

// Server regroupe l'état nécessaire aux routes du module.
type Server struct {
	Store   *config.Store
	Log     *logging.Logger
	Proxy   *proxy.Proxy               // nil = mode « ligne de code »
	Now     func() time.Time           // horloge injectable (time.Now par défaut)
	Restore func(ds []config.Deadline) // réactivation des services après renouvellement
	Changed func()                     // appelée après un enregistrement de la configuration
	License *license.Manager           // liaison licence / poste (nil = fonction absente)

	sessMu   sync.Mutex
	sessions map[string]time.Time
	failMu   sync.Mutex
	fails    map[string][]time.Time
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) status(c config.Config) scheduler.Status {
	return scheduler.ComputeStatus(c, s.Store.State(), s.now())
}

// Handler aiguille les requêtes : routes du module, page « accès suspendu », puis application.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := s.Store.Config()
		ap := c.AdminPath
		if r.URL.Path == ap || strings.HasPrefix(r.URL.Path, ap+"/") {
			s.moduleRoutes(w, r, c)
			return
		}
		st := s.status(c)
		if scheduler.Blocks(c, st, r.Host, r.URL.Path) {
			proxy.RenderExpired(w, st)
			return
		}
		if s.Proxy == nil {
			if r.URL.Path == "/" {
				http.Redirect(w, r, ap+"/admin", http.StatusFound)
				return
			}
			http.NotFound(w, r)
			return
		}
		s.Proxy.ServeHTTP(w, r)
	})
}

func (s *Server) moduleRoutes(w http.ResponseWriter, r *http.Request, c config.Config) {
	sub := strings.TrimPrefix(r.URL.Path, c.AdminPath)
	switch sub {
	case "/banner.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(web.BannerJS)
	case "/api/status": // public : utilisé par le bandeau
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-store")
		st := s.status(c)
		if p := r.URL.Query().Get("page"); p != "" {
			if pu, err := url.Parse(p); err == nil {
				st.Blocked = scheduler.Blocks(c, st, pu.Host, pu.Path)
			}
		}
		st.Contact = c.SupplierContact
		WriteJSON(w, st)
	case "/login":
		s.handleLogin(w, r, c)
	case "/logout":
		s.handleLogout(w, r, c)
	case "", "/":
		http.Redirect(w, r, c.AdminPath+"/admin", http.StatusFound)
	case "/admin":
		if !s.validSession(r) {
			s.renderLogin(w, c, c.AdminUser, "", http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		_, _ = w.Write(web.AdminHTML)
	case "/api/config":
		if !s.auth(w, r) {
			return
		}
		s.apiConfig(w, r, c)
	case "/api/restore":
		if !s.auth(w, r) || !postOnly(w, r) {
			return
		}
		ds, still := s.restorable(c)
		if s.Restore != nil && len(ds) > 0 {
			s.Restore(ds)
		}
		WriteJSON(w, map[string]any{"ok": true, "restored": len(ds), "still_stopped": still})
	case "/api/license":
		if !s.auth(w, r) {
			return
		}
		s.apiLicense(w, r)
	case "/api/log":
		if !s.auth(w, r) {
			return
		}
		WriteJSON(w, s.Log.Tail(200))
	default:
		http.NotFound(w, r)
	}
}

type adminView struct {
	config.Config
	AdminPasswordHash string               `json:"admin_password_hash,omitempty"` // masque le hash
	Status            scheduler.Status     `json:"status"`
	Proxy             bool                 `json:"proxy"`
	DefaultMessages   map[string][2]string `json:"default_messages"` // messages par défaut par type d'échéance
}

func (s *Server) apiConfig(w http.ResponseWriter, r *http.Request, c config.Config) {
	switch r.Method {
	case http.MethodGet:
		WriteJSON(w, adminView{Config: c, Status: s.status(c), Proxy: s.Proxy != nil, DefaultMessages: defaultMessages()})
	case http.MethodPost:
		if !postOnly(w, r) {
			return
		}
		var in struct {
			config.Config
			NewPassword string `json:"new_password"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "données invalides", http.StatusBadRequest)
			return
		}
		before := c
		err := s.Store.UpdateConfig(func(x *config.Config) error {
			x.ModuleName = strings.TrimSpace(in.ModuleName)
			x.SoftwareName = strings.TrimSpace(in.SoftwareName)
			x.SupplierContact = strings.TrimSpace(in.SupplierContact)
			x.Enabled = in.Enabled
			x.Deadlines = in.Deadlines
			if x.ModuleName == "" {
				x.ModuleName = "SmartGUARD"
			}
			if in.NewPassword != "" {
				if len(in.NewPassword) < 8 {
					return fmt.Errorf("le mot de passe doit contenir au moins 8 caractères")
				}
				h, err := config.HashPassword(in.NewPassword)
				if err != nil {
					return err
				}
				x.AdminPasswordHash = h
			}
			return nil
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		after := s.Store.Config()
		s.Log.Printf("Configuration modifiée par %s : logiciel « %s », %d échéance(s), module %s",
			clientIP(r), after.SoftwareName, len(after.Deadlines), map[bool]string{true: "ACTIVÉ", false: "désactivé"}[after.Enabled])
		for _, d := range after.Deadlines {
			s.Log.Printf("  échéance %s « %s » (%s) : %s → %s, arrêt à la date de fin : %v",
				d.ID, d.Name(), config.KindLabel(d.Kind), orDash(d.StartDate), d.EndDate, d.StopOnEnd)
		}
		if before.Enabled != after.Enabled {
			s.Log.Printf("Module %s", map[bool]string{true: "activé", false: "désactivé"}[after.Enabled])
		}
		if s.Changed != nil {
			go s.Changed()
		}
		WriteJSON(w, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// apiLicense : consultation (GET) puis activation / désactivation (POST) de la licence du poste.
// La clé complète n'est jamais renvoyée ni journalisée : seule sa version masquée.
func (s *Server) apiLicense(w http.ResponseWriter, r *http.Request) {
	if s.License == nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		WriteJSON(w, s.License.Status())
	case http.MethodPost:
		if !postOnly(w, r) {
			return
		}
		var in struct {
			Action string `json:"action"` // activate | deactivate
			Key    string `json:"key"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&in); err != nil {
			http.Error(w, "données invalides", http.StatusBadRequest)
			return
		}
		var err error
		switch in.Action {
		case "activate":
			var st license.Status
			if st, err = s.License.Activate(in.Key); err == nil {
				s.Log.Printf("Licence %s activée sur le poste %s par %s", st.MaskedKey, st.MachineID, clientIP(r))
			}
		case "deactivate":
			var rec license.Record
			if rec, err = s.License.Deactivate(); err == nil {
				s.Log.Printf("Licence %s désactivée (poste %s) par %s", license.Mask(rec.Key), rec.MachineID, clientIP(r))
			}
		default:
			err = fmt.Errorf("action inconnue")
		}
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		WriteJSON(w, s.License.Status())
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// restorable renvoie les échéances dont les services peuvent être relancés
// (échéances non arrêtées, c'est-à-dire renouvelées ou sans arrêt) et le
// nombre d'échéances encore arrêtées, dont les services restent arrêtés.
func (s *Server) restorable(c config.Config) ([]config.Deadline, int) {
	st := s.status(c)
	var ds []config.Deadline
	still := 0
	for i, d := range c.Deadlines {
		if st.Deadlines[i].Stopped && c.Enabled {
			still++
			continue
		}
		ds = append(ds, d)
	}
	return ds, still
}

func defaultMessages() map[string][2]string {
	m := map[string][2]string{}
	for _, k := range []string{config.KindContrat, config.KindLicence, config.KindAbonnement} {
		a, b := config.DefaultMessages(k)
		m[k] = [2]string{a, b}
	}
	return m
}

func orDash(v string) string {
	if v == "" {
		return "–"
	}
	return v
}

func postOnly(w http.ResponseWriter, r *http.Request) bool {
	// En-tête personnalisé exigé : protège contre les requêtes inter-sites (CSRF).
	if r.Method != http.MethodPost || r.Header.Get("X-SmartGUARD") != "1" {
		http.Error(w, "requête refusée", http.StatusForbidden)
		return false
	}
	return true
}

// ---- Authentification par page de connexion + cookie de session

const (
	sessionTTL    = 8 * time.Hour
	sessionCookie = "smartguard_session"
	maxFails      = 10
	failWindow    = 15 * time.Minute
)

func (s *Server) newSession() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	if s.sessions == nil {
		s.sessions = map[string]time.Time{}
	}
	now := time.Now()
	for k, exp := range s.sessions {
		if now.After(exp) {
			delete(s.sessions, k)
		}
	}
	s.sessions[tok] = now.Add(sessionTTL)
	return tok
}

func (s *Server) validSession(r *http.Request) bool {
	ck, err := r.Cookie(sessionCookie)
	if err != nil || ck.Value == "" {
		return false
	}
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	exp, ok := s.sessions[ck.Value]
	if !ok || time.Now().After(exp) {
		delete(s.sessions, ck.Value)
		return false
	}
	s.sessions[ck.Value] = time.Now().Add(sessionTTL) // prolonge la session active
	return true
}

// auth protège les API : renvoie 401 JSON (la page d'administration redirige alors vers la connexion).
func (s *Server) auth(w http.ResponseWriter, r *http.Request) bool {
	if s.validSession(r) {
		return true
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "Session expirée : reconnectez-vous."})
	return false
}

type loginView struct {
	ModuleName   string
	SoftwareName string
	Error        string
	User         string
	NoPassword   bool
}

var loginTpl = template.Must(template.New("login").Parse(web.LoginTpl))

func (s *Server) renderLogin(w http.ResponseWriter, c config.Config, user, msg string, code int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(code)
	_ = loginTpl.Execute(w, loginView{ModuleName: c.ModuleName, SoftwareName: c.SoftwareName, Error: msg, User: user, NoPassword: c.AdminPasswordHash == ""})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request, c config.Config) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, c.AdminPath+"/admin", http.StatusSeeOther)
		return
	}
	ip := clientIP(r)
	if s.tooManyFails(ip) {
		s.renderLogin(w, c, "", "Trop de tentatives. Réessayez dans 15 minutes.", http.StatusTooManyRequests)
		return
	}
	_ = r.ParseForm()
	u := strings.TrimSpace(r.PostFormValue("username"))
	p := r.PostFormValue("password")
	if c.AdminPasswordHash != "" && subtle.ConstantTimeCompare([]byte(strings.ToLower(u)), []byte(strings.ToLower(c.AdminUser))) == 1 && config.CheckPassword(c.AdminPasswordHash, p) {
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: s.newSession(), Path: c.AdminPath,
			HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
		s.Log.Printf("Connexion administrateur depuis %s", ip)
		http.Redirect(w, r, c.AdminPath+"/admin", http.StatusSeeOther)
		return
	}
	s.recordFail(ip)
	s.renderLogin(w, c, u, "Identifiant ou mot de passe incorrect.", http.StatusUnauthorized)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request, c config.Config) {
	if ck, err := r.Cookie(sessionCookie); err == nil {
		s.sessMu.Lock()
		delete(s.sessions, ck.Value)
		s.sessMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: c.AdminPath, MaxAge: -1})
	http.Redirect(w, r, c.AdminPath+"/admin", http.StatusSeeOther)
}

func (s *Server) tooManyFails(ip string) bool {
	s.failMu.Lock()
	defer s.failMu.Unlock()
	if s.fails == nil {
		s.fails = map[string][]time.Time{}
	}
	cut := time.Now().Add(-failWindow)
	keep := s.fails[ip][:0]
	for _, t := range s.fails[ip] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	s.fails[ip] = keep
	return len(keep) >= maxFails
}

func (s *Server) recordFail(ip string) {
	s.failMu.Lock()
	if s.fails == nil {
		s.fails = map[string][]time.Time{}
	}
	s.fails[ip] = append(s.fails[ip], time.Now())
	s.failMu.Unlock()
	s.Log.Printf("Échec d'authentification administrateur depuis %s", ip)
}

func clientIP(r *http.Request) string {
	return scheduler.HostOnly(r.RemoteAddr)
}

// WriteJSON envoie une réponse JSON.
func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
