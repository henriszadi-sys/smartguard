package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smartguard/internal/license"
)

func TestLicenseAPI(t *testing.T) {
	s := newServer(t)
	mid := "poste-a"
	s.License = &license.Manager{
		Path:    filepath.Join(t.TempDir(), "config.license.json"),
		Now:     func() time.Time { return time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local) },
		Machine: func() (string, error) { return mid, nil },
	}
	key := license.MakeKey("ABCD", "EFGH", "JKLM")

	// Sans session : refusé.
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/_smartguard/api/license", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("API licence sans session : code %d", w.Code)
	}

	cookie := login(s, "motdepasse1").Result().Cookies()[0]
	call := func(method, body string, header bool) (int, license.Status, string) {
		r := httptest.NewRequest(method, "/_smartguard/api/license", strings.NewReader(body))
		r.AddCookie(cookie)
		if header {
			r.Header.Set("X-SmartGUARD", "1")
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		var st license.Status
		_ = json.Unmarshal(w.Body.Bytes(), &st)
		return w.Code, st, w.Body.String()
	}

	if code, st, _ := call(http.MethodGet, "", false); code != http.StatusOK || st.State != license.StateNone {
		t.Fatalf("état initial : %d %+v", code, st)
	}
	if code, _, _ := call(http.MethodPost, `{"action":"activate","key":"`+key+`"}`, false); code != http.StatusForbidden {
		t.Fatalf("POST sans en-tête anti-CSRF : code %d", code)
	}
	if code, _, body := call(http.MethodPost, `{"action":"activate","key":"SGRD-AAAA-BBBB-CCCC-DDDD"}`, true); code != http.StatusBadRequest || !strings.Contains(body, "invalide") {
		t.Fatalf("clé invalide : %d %s", code, body)
	}

	code, st, body := call(http.MethodPost, `{"action":"activate","key":"`+strings.ToLower(key)+`"}`, true)
	if code != http.StatusOK || st.State != license.StateActive {
		t.Fatalf("activation : %d %s", code, body)
	}
	if strings.Contains(body, "ABCD") || strings.Contains(readLog(t, s), "ABCD") || strings.Contains(readLog(t, s), "JKLM") {
		t.Fatal("la clé complète apparaît dans la réponse ou le journal")
	}
	if !strings.Contains(readLog(t, s), "activée sur le poste poste-a") {
		t.Fatalf("activation non journalisée : %s", readLog(t, s))
	}

	// Transfert : désactivation, puis activation sur un autre poste.
	if code, st, _ := call(http.MethodPost, `{"action":"deactivate"}`, true); code != http.StatusOK || st.State != license.StateNone {
		t.Fatalf("désactivation : %d %+v", code, st)
	}
	mid = "poste-b"
	if code, st, _ := call(http.MethodPost, `{"action":"activate","key":"`+key+`"}`, true); code != http.StatusOK || st.MachineID != "poste-b" {
		t.Fatalf("activation sur le nouveau poste : %d %+v", code, st)
	}
	if code, _, _ := call(http.MethodPost, `{"action":"inconnue"}`, true); code != http.StatusBadRequest {
		t.Fatalf("action inconnue : code %d", code)
	}
}
