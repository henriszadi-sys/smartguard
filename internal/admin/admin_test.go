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
		c.Enabled, c.StartDate, c.EndDate, c.AdminPasswordHash = true, "2026-01-01", "2026-10-05", h
		c.BlockedURLs = []string{"/"}
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
	s.Restore = func(config.Config) { restored++ }
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
		t.Fatal("page bloquée avant l'échéance")
	}
	s.Now = func() time.Time { return time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local) }
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "</html>") {
		t.Fatalf("page « accès suspendu » attendue à l'échéance : code %d", w.Code)
	}
}
