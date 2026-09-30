package actions

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// counterScript renvoie une commande qui ajoute une ligne au fichier à chaque exécution,
// puis se termine avec le code indiqué.
func counterScript(file, code string) string {
	if runtime.GOOS == "windows" {
		return `echo x>>"` + file + `" & exit ` + code
	}
	return `echo x >> '` + file + `'; exit ` + code
}

// tempFile renvoie un chemin dans un dossier dont le nom contient des espaces
// (cas de « C:\Program Files »).
func tempFile(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "dossier avec espaces")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "count.txt")
}

func runs(t *testing.T, file string) int {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "x")
}

func TestScriptRetriedThreeTimesThenFails(t *testing.T) {
	retryDelay = 0
	file := tempFile(t)
	res := RunScript(counterScript(file, "1"))
	if got := runs(t, file); got != ScriptAttempts {
		t.Fatalf("script lancé %d fois, attendu %d", got, ScriptAttempts)
	}
	if !strings.HasPrefix(res, "ÉCHEC") || !strings.Contains(res, "après 3 tentatives") {
		t.Fatalf("échec mal journalisé : %q", res)
	}
}

func TestSuccessfulScriptRunsOnce(t *testing.T) {
	retryDelay = 0
	file := tempFile(t)
	res := RunScript(counterScript(file, "0"))
	if got := runs(t, file); got != 1 {
		t.Fatalf("script lancé %d fois, attendu 1", got)
	}
	if !strings.HasPrefix(res, "OK") {
		t.Fatalf("résultat : %q", res)
	}
}
