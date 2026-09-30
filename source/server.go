package main

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed assets/banner.js
var bannerJS []byte

//go:embed assets/admin.html
var adminHTML []byte

//go:embed assets/login.html
var loginTpl string

//go:embed assets/expired.html
var expiredTpl string

type App struct {
	store    *Store
	logPath  string
	logMu    sync.Mutex
	proxy    *httputil.ReverseProxy
	target   *url.URL
	fails    map[string][]time.Time
	sessions map[string]time.Time
	sessMu   sync.Mutex
	failMu   sync.Mutex
}

func (a *App) logf(format string, args ...any) {
	line := time.Now().Format("2006-01-02 15:04:05") + "  " + fmt.Sprintf(format, args...)
	fmt.Println(line)
	a.logMu.Lock()
	defer a.logMu.Unlock()
	f, err := os.OpenFile(a.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		fmt.Fprintln(f, line)
		f.Close()
	}
}

func (a *App) tailLog(n int) []string {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	b, err := os.ReadFile(a.logPath)
	if err != nil {
		return []string{}
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines
}

func (a *App) setupProxy(upstream string) error {
	if upstream == "" {
		return nil
	}
	u, err := url.Parse(upstream)
	if err != nil || u.Host == "" {
		return fmt.Errorf("adresse de l'application (upstream) invalide : %q", upstream)
	}
	a.target = u
	a.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.SetXForwarded()
			pr.Out.Host = pr.In.Host             // l'application voit l'adresse publique
			pr.Out.Header.Del("Accept-Encoding") // réponse non compressée pour injecter le bandeau
		},
		ModifyResponse: a.injectBanner,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			a.logf("proxy : application injoignable (%s) : %v", upstream, err)
			http.Error(w, "Application momentanément indisponible.", http.StatusBadGateway)
		},
	}
	return nil
}

var bodyClose = regexp.MustCompile(`(?i)</body\s*>`)

func (a *App) injectBanner(resp *http.Response) error {
	// Réécrit les redirections qui pointeraient vers l'adresse interne.
	if loc := resp.Header.Get("Location"); loc != "" && a.target != nil {
		if lu, err := url.Parse(loc); err == nil && strings.EqualFold(lu.Host, a.target.Host) {
			lu.Scheme, lu.Host = "", ""
			resp.Header.Set("Location", lu.String())
		}
	}
	c := a.store.Config()
	if !c.Enabled {
		return nil
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(ct, "text/html") || resp.Header.Get("Content-Encoding") != "" {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	resp.Body.Close()
	if err != nil {
		return err
	}
	tag := []byte(`<script src="` + c.AdminPath + `/banner.js" defer></script>`)
	if locs := bodyClose.FindAllIndex(body, -1); len(locs) > 0 {
		i := locs[len(locs)-1][0]
		body = append(body[:i:i], append(tag, body[i:]...)...)
	} else if bytes.Contains(bytes.ToLower(body[:min(len(body), 2048)]), []byte("<html")) {
		body = append(body, tag...)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	// La politique CSP de l'application pourrait interdire le script : on l'autorise sur la même origine.
	return nil
}

func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := a.store.Config()
		ap := c.AdminPath
		if r.URL.Path == ap || strings.HasPrefix(r.URL.Path, ap+"/") {
			a.moduleRoutes(w, r, c)
			return
		}
		st := computeStatus(c, a.store.State())
		if c.Enabled && st.Expired && isBlocked(c, r.Host, r.URL.Path) {
			a.renderExpired(w, st)
			return
		}
		if a.proxy == nil {
			if r.URL.Path == "/" {
				http.Redirect(w, r, ap+"/admin", http.StatusFound)
				return
			}
			http.NotFound(w, r)
			return
		}
		a.proxy.ServeHTTP(w, r)
	})
}

func (a *App) renderExpired(w http.ResponseWriter, st Status) {
	t := template.Must(template.New("x").Parse(expiredTpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = t.Execute(w, st)
}

func (a *App) moduleRoutes(w http.ResponseWriter, r *http.Request, c Config) {
	sub := strings.TrimPrefix(r.URL.Path, c.AdminPath)
	switch sub {
	case "/banner.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(bannerJS)
	case "/api/status": // public : utilisé par le bandeau
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-store")
		st := computeStatus(c, a.store.State())
		if st.Expired && c.Enabled {
			if p := r.URL.Query().Get("page"); p != "" {
				if pu, err := url.Parse(p); err == nil {
					st.Blocked = isBlocked(c, pu.Host, pu.Path)
				}
			}
		}
		st.Contact = c.SupplierContact
		writeJSONResp(w, st)
	case "/login":
		a.handleLogin(w, r, c)
	case "/logout":
		a.handleLogout(w, r, c)
	case "", "/":
		http.Redirect(w, r, c.AdminPath+"/admin", http.StatusFound)
	case "/admin":
		if !a.validSession(r) {
			a.renderLogin(w, c, c.AdminUser, "", http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		_, _ = w.Write(adminHTML)
	case "/api/config":
		if !a.auth(w, r, c) {
			return
		}
		a.apiConfig(w, r, c)
	case "/api/restore":
		if !a.auth(w, r, c) || !postOnly(w, r) {
			return
		}
		a.restoreAfterRenewal(c)
		writeJSONResp(w, map[string]any{"ok": true})
	case "/api/log":
		if !a.auth(w, r, c) {
			return
		}
		writeJSONResp(w, a.tailLog(200))
	default:
		http.NotFound(w, r)
	}
}

type adminView struct {
	Config
	AdminPasswordHash string `json:"admin_password_hash,omitempty"` // masque le hash
	Status            Status `json:"status"`
	ActionsDoneAt     string `json:"actions_done_at"`
	Proxy             bool   `json:"proxy"`
}

func (a *App) apiConfig(w http.ResponseWriter, r *http.Request, c Config) {
	switch r.Method {
	case http.MethodGet:
		st := a.store.State()
		v := adminView{Config: c, Status: computeStatus(c, st), Proxy: a.proxy != nil}
		if !st.ActionsDoneAt.IsZero() && st.ActionsFor == c.EndDate {
			v.ActionsDoneAt = st.ActionsDoneAt.Format("02/01/2006 15:04")
		}
		writeJSONResp(w, v)
	case http.MethodPost:
		if !postOnly(w, r) {
			return
		}
		var in struct {
			Config
			NewPassword string `json:"new_password"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "données invalides", http.StatusBadRequest)
			return
		}
		before := c
		err := a.store.UpdateConfig(func(x *Config) error {
			x.ModuleName = strings.TrimSpace(in.ModuleName)
			x.SoftwareName = strings.TrimSpace(in.SoftwareName)
			x.SupplierContact = strings.TrimSpace(in.SupplierContact)
			x.Enabled = in.Enabled
			x.StartDate, x.EndDate = in.StartDate, in.EndDate
			x.WarningDays = in.WarningDays
			x.Message, x.ExpiredMessage = in.Message, in.ExpiredMessage
			x.Services, x.BlockedURLs, x.Scripts = in.Services, in.BlockedURLs, in.Scripts
			if x.ModuleName == "" {
				x.ModuleName = "SmartGUARD"
			}
			if in.NewPassword != "" {
				if len(in.NewPassword) < 8 {
					return fmt.Errorf("le mot de passe doit contenir au moins 8 caractères")
				}
				h, err := hashPassword(in.NewPassword)
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
		after := a.store.Config()
		a.logf("Configuration modifiée par %s : logiciel « %s », %s → %s, module %s",
			clientIP(r), after.SoftwareName, after.StartDate, after.EndDate, map[bool]string{true: "ACTIVÉ", false: "désactivé"}[after.Enabled])
		if before.Enabled != after.Enabled {
			a.logf("Module %s", map[bool]string{true: "activé", false: "désactivé"}[after.Enabled])
		}
		go a.tick()
		writeJSONResp(w, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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

const sessionTTL = 8 * time.Hour

func (a *App) newSession() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	now := time.Now()
	for k, exp := range a.sessions {
		if now.After(exp) {
			delete(a.sessions, k)
		}
	}
	a.sessions[tok] = now.Add(sessionTTL)
	return tok
}

func (a *App) validSession(r *http.Request) bool {
	ck, err := r.Cookie("smartguard_session")
	if err != nil || ck.Value == "" {
		return false
	}
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	exp, ok := a.sessions[ck.Value]
	if !ok || time.Now().After(exp) {
		delete(a.sessions, ck.Value)
		return false
	}
	a.sessions[ck.Value] = time.Now().Add(sessionTTL) // prolonge la session active
	return true
}

// auth protège les API : renvoie 401 JSON (la page d'administration redirige alors vers la connexion).
func (a *App) auth(w http.ResponseWriter, r *http.Request, c Config) bool {
	if a.validSession(r) {
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

func (a *App) renderLogin(w http.ResponseWriter, c Config, user, msg string, code int) {
	t := template.Must(template.New("l").Parse(loginTpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(code)
	_ = t.Execute(w, loginView{ModuleName: c.ModuleName, SoftwareName: c.SoftwareName, Error: msg, User: user, NoPassword: c.AdminPasswordHash == ""})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request, c Config) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, c.AdminPath+"/admin", http.StatusSeeOther)
		return
	}
	ip := clientIP(r)
	if a.tooManyFails(ip) {
		a.renderLogin(w, c, "", "Trop de tentatives. Réessayez dans 15 minutes.", http.StatusTooManyRequests)
		return
	}
	_ = r.ParseForm()
	u := strings.TrimSpace(r.PostFormValue("username"))
	p := r.PostFormValue("password")
	if c.AdminPasswordHash != "" && subtle.ConstantTimeCompare([]byte(strings.ToLower(u)), []byte(strings.ToLower(c.AdminUser))) == 1 && checkPassword(c.AdminPasswordHash, p) {
		http.SetCookie(w, &http.Cookie{Name: "smartguard_session", Value: a.newSession(), Path: c.AdminPath,
			HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
		a.logf("Connexion administrateur depuis %s", ip)
		http.Redirect(w, r, c.AdminPath+"/admin", http.StatusSeeOther)
		return
	}
	a.recordFail(ip)
	a.renderLogin(w, c, u, "Identifiant ou mot de passe incorrect.", http.StatusUnauthorized)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request, c Config) {
	if ck, err := r.Cookie("smartguard_session"); err == nil {
		a.sessMu.Lock()
		delete(a.sessions, ck.Value)
		a.sessMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "smartguard_session", Value: "", Path: c.AdminPath, MaxAge: -1})
	http.Redirect(w, r, c.AdminPath+"/admin", http.StatusSeeOther)
}

func (a *App) tooManyFails(ip string) bool {
	a.failMu.Lock()
	defer a.failMu.Unlock()
	cut := time.Now().Add(-15 * time.Minute)
	keep := a.fails[ip][:0]
	for _, t := range a.fails[ip] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	a.fails[ip] = keep
	return len(keep) >= 10
}

func (a *App) recordFail(ip string) {
	a.failMu.Lock()
	a.fails[ip] = append(a.fails[ip], time.Now())
	a.failMu.Unlock()
	a.logf("Échec d'authentification administrateur depuis %s", ip)
}

func clientIP(r *http.Request) string {
	return hostOnly(r.RemoteAddr)
}

func writeJSONResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
