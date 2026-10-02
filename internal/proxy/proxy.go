// Package proxy place le module devant l'application (mode automatique) :
// il relaie les requêtes, injecte le bandeau dans les pages HTML et affiche
// la page « accès suspendu » après l'échéance.
package proxy

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"smartguard/internal/config"
	"smartguard/internal/scheduler"
	"smartguard/web"
)

// Proxy relaie les requêtes vers l'application surveillée.
type Proxy struct {
	rp     *httputil.ReverseProxy
	target *url.URL
	store  *config.Store
}

// New prépare le proxy ; renvoie nil sans erreur si aucune application n'est configurée
// (mode « ligne de code »).
func New(upstream string, store *config.Store, logf func(format string, args ...any)) (*Proxy, error) {
	if upstream == "" {
		return nil, nil
	}
	u, err := url.Parse(upstream)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("adresse de l'application (upstream) invalide : %q", upstream)
	}
	p := &Proxy{target: u, store: store}
	p.rp = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.SetXForwarded()
			pr.Out.Host = pr.In.Host             // l'application voit l'adresse publique
			pr.Out.Header.Del("Accept-Encoding") // réponse non compressée pour injecter le bandeau
		},
		ModifyResponse: p.injectBanner,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logf("proxy : application injoignable (%s) : %v", upstream, err)
			http.Error(w, "Application momentanément indisponible.", http.StatusBadGateway)
		},
	}
	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) { p.rp.ServeHTTP(w, r) }

var bodyClose = regexp.MustCompile(`(?i)</body\s*>`)

func (p *Proxy) injectBanner(resp *http.Response) error {
	// Réécrit les redirections qui pointeraient vers l'adresse interne.
	if loc := resp.Header.Get("Location"); loc != "" {
		if lu, err := url.Parse(loc); err == nil && strings.EqualFold(lu.Host, p.target.Host) {
			lu.Scheme, lu.Host = "", ""
			resp.Header.Set("Location", lu.String())
		}
	}
	c := p.store.Config()
	if !c.Enabled && !c.ReportEnabled { // ni rappel ni lien « Signaler un problème »
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
	return nil
}

var expiredTpl = template.Must(template.New("expired").Parse(web.ExpiredTpl))

// RenderExpired affiche la page « accès suspendu ».
func RenderExpired(w http.ResponseWriter, st scheduler.Status) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = expiredTpl.Execute(w, st)
}
