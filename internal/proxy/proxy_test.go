package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"smartguard/internal/config"
)

func TestInjectsBannerAndRewritesRedirects(t *testing.T) {
	var app *httptest.Server
	app = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/old":
			http.Redirect(w, r, app.URL+"/new", http.StatusFound)
		case "/data":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"a":"</body>"}`)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<html><body><h1>Appli</h1></body></html>")
		}
	}))
	defer app.Close()

	st, err := config.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateConfig(func(c *config.Config) error {
		c.Enabled, c.Deadlines = true, []config.Deadline{{EndDate: "2026-10-05"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	p, err := New(app.URL, st, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(p)
	defer front.Close()
	cl := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	body := get(t, cl, front.URL+"/")
	if !strings.Contains(body, `<script src="/_smartguard/banner.js" defer></script></body>`) {
		t.Fatalf("bandeau non injecté : %s", body)
	}
	if body := get(t, cl, front.URL+"/data"); strings.Contains(body, "banner.js") {
		t.Fatal("bandeau injecté dans une réponse non HTML")
	}
	resp, err := cl.Get(front.URL + "/old")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); loc != "/new" {
		t.Fatalf("redirection vers l'adresse interne non réécrite : %q", loc)
	}

	if err := st.UpdateConfig(func(c *config.Config) error { c.Enabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if body := get(t, cl, front.URL+"/"); strings.Contains(body, "banner.js") {
		t.Fatal("module désactivé : bandeau injecté")
	}
	// Module désactivé mais lien « Signaler un problème » activé : le script reste injecté.
	if err := st.UpdateConfig(func(c *config.Config) error {
		c.ReportEnabled, c.ReportURL = true, "https://portail.exemple.ci/signaler"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if body := get(t, cl, front.URL+"/"); !strings.Contains(body, "banner.js") {
		t.Fatal("lien « Signaler un problème » activé : script non injecté")
	}
}

func get(t *testing.T, cl *http.Client, url string) string {
	t.Helper()
	resp, err := cl.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}
