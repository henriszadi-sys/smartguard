package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"smartguard/internal/config"
)

func loginAs(s *Server, user, pw string) *httptest.ResponseRecorder {
	form := url.Values{"username": {user}, "password": {pw}}
	r := httptest.NewRequest(http.MethodPost, "/_smartguard/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.RemoteAddr = "10.0.0.9:5000"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func call(s *Server, ck *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/_smartguard"+path, strings.NewReader(body))
	if ck != nil {
		r.AddCookie(ck)
	}
	if method == http.MethodPost {
		r.Header.Set("X-SmartGUARD", "1")
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

// L'accès client est désactivé tant que le technicien n'a pas défini son mot de passe ;
// il permet la consultation et la réactivation, jamais les réglages.
func TestClientAccessIsReadOnlyPlusRestore(t *testing.T) {
	s := newServer(t)
	s.Now = func() time.Time { return time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local) }
	if w := loginAs(s, "client", "motdepasse1"); w.Code != http.StatusUnauthorized {
		t.Fatalf("accès client actif par défaut : code %d", w.Code)
	}
	tech := login(s, "motdepasse1").Result().Cookies()[0]

	// Le mot de passe client ne peut pas être celui du technicien.
	if w := call(s, tech, http.MethodPost, "/api/config", `{"module_name":"Kelio","enabled":true,"deadlines":[{"id":"d1","end_date":"2026-10-05","stop_on_end":true,"services":["kelio"],"blocked_urls":["/"]}],"new_client_password":"motdepasse1"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("mot de passe client identique accepté : code %d", w.Code)
	}
	if w := call(s, tech, http.MethodPost, "/api/config", `{"module_name":"Kelio","enabled":true,"deadlines":[{"id":"d1","end_date":"2026-10-05","stop_on_end":true,"services":["kelio"],"blocked_urls":["/"]}],"new_client_password":"clientpass2"}`); w.Code != http.StatusOK {
		t.Fatalf("création de l'accès client : code %d %s", w.Code, w.Body.String())
	}
	lw := loginAs(s, "client", "clientpass2")
	if lw.Code != http.StatusSeeOther {
		t.Fatalf("connexion client : code %d", lw.Code)
	}
	cli := lw.Result().Cookies()[0]

	// Consultation : autorisée, rôle indiqué, aucun haché de mot de passe.
	w := call(s, cli, http.MethodGet, "/api/config", "")
	var v map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	if w.Code != http.StatusOK || v["role"] != RoleClient || v["client_enabled"] != true {
		t.Fatalf("consultation client : code %d %v", w.Code, v["role"])
	}
	if strings.Contains(w.Body.String(), "pbkdf2") {
		t.Fatal("haché de mot de passe envoyé à l'interface")
	}
	// Réglages : refusés (403) ; configuration inchangée.
	for _, c := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/config", `{"module_name":"Pirate","deadlines":[]}`},
		{http.MethodPost, "/api/license", `{"action":"deactivate"}`},
		{http.MethodGet, "/api/export", ""},
		{http.MethodPost, "/api/import", `{}`},
	} {
		if w := call(s, cli, c.method, c.path, c.body); w.Code != http.StatusForbidden {
			t.Errorf("accès client %s %s : code %d, attendu 403", c.method, c.path, w.Code)
		}
	}
	if s.Store.Config().ModuleName != "Kelio" {
		t.Fatal("configuration modifiée par l'accès client")
	}
	// Journal et réactivation : autorisés.
	if w := call(s, cli, http.MethodGet, "/api/log", ""); w.Code != http.StatusOK {
		t.Fatalf("journal client : code %d", w.Code)
	}
	if w := call(s, cli, http.MethodPost, "/api/restore", `{}`); w.Code != http.StatusOK {
		t.Fatalf("réactivation client : code %d", w.Code)
	}
	// Le technicien désactive l'accès client : nouvelle connexion refusée.
	if w := call(s, tech, http.MethodPost, "/api/config", `{"module_name":"Kelio","enabled":true,"deadlines":[{"id":"d1","end_date":"2026-10-05"}],"client_enabled":false}`); w.Code != http.StatusOK {
		t.Fatalf("désactivation de l'accès client : code %d", w.Code)
	}
	if w := loginAs(s, "client", "clientpass2"); w.Code != http.StatusUnauthorized {
		t.Fatalf("accès client désactivé mais connexion acceptée : code %d", w.Code)
	}
}

// Sauvegarde : l'export ne contient aucun secret ; l'import reprend les échéances
// sans toucher au port ni aux mots de passe.
func TestExportImport(t *testing.T) {
	s := newServer(t)
	tech := login(s, "motdepasse1").Result().Cookies()[0]
	w := call(s, tech, http.MethodGet, "/api/export", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Disposition"), ".smartguard.json") {
		t.Fatalf("export : code %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "pbkdf2") {
		t.Fatal("l'export contient un haché de mot de passe")
	}
	var e Export
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil || len(e.Config.Deadlines) != 1 {
		t.Fatalf("export illisible : %v", err)
	}
	// Sauvegarde modifiée : autre port, autre échéance.
	e.Config.Listen = ":9999"
	e.Config.Deadlines = append(e.Config.Deadlines, config.Deadline{Kind: config.KindLicence, EndDate: "2027-12-31"})
	b, _ := json.Marshal(e)
	before := s.Store.Config()
	if w := call(s, tech, http.MethodPost, "/api/import", string(b)); w.Code != http.StatusOK {
		t.Fatalf("import : code %d %s", w.Code, w.Body.String())
	}
	after := s.Store.Config()
	if len(after.Deadlines) != 2 || after.Listen != before.Listen || after.AdminPasswordHash != before.AdminPasswordHash {
		t.Fatalf("import : échéances %d, port %s, mot de passe conservé %v", len(after.Deadlines), after.Listen, after.AdminPasswordHash == before.AdminPasswordHash)
	}
	if w := call(s, tech, http.MethodPost, "/api/import", `{"format":"autre"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("fichier étranger importé : code %d", w.Code)
	}
}
