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
	// Arrêt exécuté pour une échéance précédente, depuis renouvelée.
	s.Store.UpdateState(func(st *config.State) {
		st.Actions = map[string]config.ActionRecord{"d1": {DoneAt: time.Date(2025, 10, 5, 0, 0, 5, 0, time.Local), For: "2025-10-05"}}
	})
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
	s.Store.UpdateState(func(st *config.State) {
		st.Actions = map[string]config.ActionRecord{"d2": {DoneAt: time.Date(2026, 1, 1, 0, 0, 5, 0, time.Local), For: "2026-01-01"}}
	})
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

// Réactivation d'une échéance précise : refusée tant qu'elle est arrêtée,
// acceptée après renouvellement, et mémorisée.
func TestRestoreSingleDeadline(t *testing.T) {
	s := newServer(t)
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.Local)
	s.Now = func() time.Time { return now }
	var got []string
	s.Restore = func(ds []config.Deadline) {
		for _, d := range ds {
			got = append(got, d.ID)
		}
	}
	if err := s.Store.UpdateConfig(func(c *config.Config) error {
		c.Deadlines = append(c.Deadlines, config.Deadline{Kind: config.KindLicence, EndDate: "2027-01-01", StopOnEnd: true, Services: []string{"paie"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Actions de d1 exécutées à l'échéance.
	s.Store.UpdateState(func(st *config.State) {
		st.Actions = map[string]config.ActionRecord{"d1": {DoneAt: time.Date(2026, 10, 5, 0, 0, 5, 0, time.Local), For: "2026-10-05"}}
	})
	cookie := login(s, "motdepasse1").Result().Cookies()[0]
	post := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/_smartguard/api/restore", strings.NewReader(body))
		r.AddCookie(cookie)
		r.Header.Set("X-SmartGUARD", "1")
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	if w := post(`{"id":"d1"}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "encore arrêtée") || len(got) != 0 {
		t.Fatalf("échéance arrêtée : code %d %s, réactivées %v", w.Code, w.Body.String(), got)
	}
	if w := post(`{"id":"inconnue"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("échéance inconnue : code %d", w.Code)
	}
	if w := post(`{"id":"d2"}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "aucun arrêt") {
		t.Fatalf("échéance jamais arrêtée : code %d %s", w.Code, w.Body.String())
	}
	// Renouvellement de d1 puis réactivation de d1 seule.
	if err := s.Store.UpdateConfig(func(c *config.Config) error { c.Deadlines[0].EndDate = "2027-10-05"; return nil }); err != nil {
		t.Fatal(err)
	}
	if w := post(`{"id":"d1"}`); w.Code != http.StatusOK || len(got) != 1 || got[0] != "d1" {
		t.Fatalf("réactivation de d1 : code %d, réactivées %v", w.Code, got)
	}
	if r := s.Store.State().Actions["d1"]; !r.RestoredAt.Equal(now) {
		t.Fatalf("réactivation non mémorisée : %+v", r)
	}
	// Réactivation globale ensuite : plus rien à relancer (d1 déjà réactivée, d2 jamais arrêtée).
	if w := post(`{}`); w.Code != http.StatusOK || len(got) != 1 || !strings.Contains(w.Body.String(), `"restored":0`) {
		t.Fatalf("réactivation globale : code %d %s, réactivées %v", w.Code, w.Body.String(), got)
	}
}

// Page bloquée : le message et le libellé sont ceux de l'échéance à l'origine du blocage ;
// l'état public ne contient pas le détail des actions.
func TestBlockedPageNamesDeadline(t *testing.T) {
	s := newServer(t)
	s.Now = func() time.Time { return time.Date(2026, 10, 6, 9, 0, 0, 0, time.Local) }
	if err := s.Store.UpdateConfig(func(c *config.Config) error {
		c.Deadlines[0].BlockedURLs = []string{"/kelio"}
		c.Deadlines[0].Label = "Maintenance Kelio"
		c.Deadlines = append(c.Deadlines, config.Deadline{Kind: config.KindLicence, Label: "Licence paie", EndDate: "2026-10-01", StopOnEnd: true, BlockedURLs: []string{"/paie"}})
		c.ReportEnabled, c.ReportURL = true, "https://portail.exemple.ci/signaler"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/paie/bulletins", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "Licence paie") || !strings.Contains(w.Body.String(), "a expiré le 01/10/2026") {
		t.Fatalf("page bloquée par la licence : code %d\n%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "https://portail.exemple.ci/signaler") {
		t.Fatal("lien « Signaler un problème » absent de la page bloquée")
	}
	r := httptest.NewRequest(http.MethodGet, "/_smartguard/api/status?page="+url.QueryEscape("http://srv/kelio/accueil"), nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `"blocked_by":"Maintenance Kelio"`) || !strings.Contains(body, `"report_url":"https://portail.exemple.ci/signaler"`) {
		t.Fatalf("état public : %s", body)
	}
	if strings.Contains(body, "actions_done_at") || strings.Contains(body, "restored_at") {
		t.Fatalf("détail des actions publié : %s", body)
	}
}
