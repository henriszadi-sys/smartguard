package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smartguard/internal/config"
	"smartguard/internal/logging"
)

func newServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	st, err := config.NewStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := config.HashPassword("motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *config.Config) error {
		c.Enabled, c.AdminPasswordHash = true, h
		c.Deadlines = []config.Deadline{{Kind: config.KindContrat, StartDate: "2026-01-01", EndDate: "2026-10-05",
			StopOnEnd: true, BlockedURLs: []string{"/"}, Services: []string{"kelio"}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return &Server{Store: st, Log: logging.New(filepath.Join(dir, "config.log"))}
}

func login(s *Server, pw string) *httptest.ResponseRecorder {
	form := url.Values{"username": {"admin"}, "password": {pw}}
	r := httptest.NewRequest(http.MethodPost, "/_smartguard/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.RemoteAddr = "10.0.0.5:5000"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func TestLoginLocksAfterTenFailures(t *testing.T) {
	s := newServer(t)
	for i := 0; i < 10; i++ {
		if w := login(s, "mauvais"); w.Code != http.StatusUnauthorized {
			t.Fatalf("échec %d : code %d", i+1, w.Code)
		}
	}
	if w := login(s, "motdepasse1"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("après 10 échecs, bon mot de passe : code %d, attendu 429", w.Code)
	}
	if strings.Contains(readLog(t, s), "motdepasse1") || strings.Contains(readLog(t, s), "mauvais") {
		t.Fatal("un mot de passe a été écrit dans le journal")
	}
}

func readLog(t *testing.T, s *Server) string {
	t.Helper()
	return strings.Join(s.Log.Tail(1000), "\n")
}

func TestProtectedAPIRequiresSessionAndHeader(t *testing.T) {
	s := newServer(t)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/_smartguard/api/config", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("API sans session : code %d", w.Code)
	}

	lw := login(s, "motdepasse1")
	if lw.Code != http.StatusSeeOther {
		t.Fatalf("connexion : code %d", lw.Code)
	}
	cookie := lw.Result().Cookies()[0]

	restored := 0
	s.Now = func() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local) }
	s.Restore = func([]config.Deadline) { restored++ }
	post := func(withHeader bool) int {
		r := httptest.NewRequest(http.MethodPost, "/_smartguard/api/restore", nil)
		r.AddCookie(cookie)
		if withHeader {
			r.Header.Set("X-SmartGUARD", "1")
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if code := post(false); code != http.StatusForbidden || restored != 0 {
		t.Fatalf("POST sans en-tête anti-CSRF : code %d, réactivations %d", code, restored)
	}
	if code := post(true); code != http.StatusOK || restored != 1 {
		t.Fatalf("réactivation explicite : code %d, réactivations %d", code, restored)
	}
}

func TestExpiredPageUsesInjectedClock(t *testing.T) {
	s := newServer(t)
	s.Now = func() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.Local) }
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code == http.StatusServiceUnavailable {
		t.Fatal("page bloquée avant la fin du contrat")
	}
	// Avec l'option d'arrêt, la page est bloquée à 00:00 le jour de la date de fin.
	s.Now = func() time.Time { return time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local) }
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "</html>") {
		t.Fatalf("page « accès suspendu » attendue à la date de fin : code %d", w.Code)
	}
	// Sans l'option d'arrêt, la fin du contrat ne bloque rien.
	if err := s.Store.UpdateConfig(func(c *config.Config) error { c.Deadlines[0].StopOnEnd = false; return nil }); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code == http.StatusServiceUnavailable {
		t.Fatal("page bloquée à la fin du contrat sans option d'arrêt")
	}
}

// Réactivation : seuls les services des échéances qui ne sont plus arrêtées sont relancés.
func TestRestoreSkipsStillStoppedDeadlines(t *testing.T) {
	s := newServer(t)
	s.Now = func() time.Time { return time.Date(2026, 10, 6, 9, 0, 0, 0, time.Local) }
	if err := s.Store.UpdateConfig(func(c *config.Config) error {
		c.Deadlines = append(c.Deadlines, config.Deadline{Kind: config.KindLicence, EndDate: "2027-01-01", StopOnEnd: true, Services: []string{"paie"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ds, still := s.restorable(s.Store.Config())
	if still != 1 || len(ds) != 1 || ds[0].Services[0] != "paie" {
		t.Fatalf("réactivation : %+v, encore arrêtées %d", ds, still)
	}
}

// Enregistrement de plusieurs échéances par l'interface d'administration.
func TestConfigAPISavesDeadlines(t *testing.T) {
	s := newServer(t)
	cookie := login(s, "motdepasse1").Result().Cookies()[0]
	body := `{"module_name":"Kelio","software_name":"Kelio","enabled":true,"deadlines":[
		{"id":"d1","kind":"contrat","start_date":"2026-01-01","end_date":"2026-12-31","stop_on_end":false},
		{"kind":"licence","label":"Licence Kelio","end_date":"2027-03-31","stop_on_end":true,"services":["kelio"]}]}`
	r := httptest.NewRequest(http.MethodPost, "/_smartguard/api/config", strings.NewReader(body))
	r.AddCookie(cookie)
	r.Header.Set("X-SmartGUARD", "1")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("enregistrement : code %d %s", w.Code, w.Body.String())
	}
	c := s.Store.Config()
	if len(c.Deadlines) != 2 || c.Deadlines[1].ID != "d2" || c.Deadlines[1].Label != "Licence Kelio" || !c.Deadlines[1].StopOnEnd {
		t.Fatalf("échéances enregistrées : %+v", c.Deadlines)
	}
	// Échéance invalide : refus et configuration inchangée.
	bad := strings.Replace(body, "2027-03-31", "", 1)
	r = httptest.NewRequest(http.MethodPost, "/_smartguard/api/config", strings.NewReader(bad))
	r.AddCookie(cookie)
	r.Header.Set("X-SmartGUARD", "1")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || len(s.Store.Config().Deadlines) != 2 {
		t.Fatalf("échéance sans date de fin : code %d", w.Code)
	}
}
