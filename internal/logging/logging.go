// Package logging écrit les journaux du module : config.log (fonctionnement)
// et installation.log (installations et mises à jour).
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger écrit dans le journal du module et sur la sortie standard.
type Logger struct {
	path string
	mu   sync.Mutex
}

func New(path string) *Logger { return &Logger{path: path} }

func (l *Logger) Path() string { return l.path }

// Printf ajoute une ligne horodatée au journal.
func (l *Logger) Printf(format string, args ...any) {
	line := time.Now().Format("2006-01-02 15:04:05") + "  " + fmt.Sprintf(format, args...)
	fmt.Println(line)
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		fmt.Fprintln(f, line)
		f.Close()
	}
}

// Tail renvoie les n dernières lignes du journal, la plus récente en premier.
func (l *Logger) Tail(n int) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, err := os.ReadFile(l.path)
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

// PathFor : chemin du journal associé à un fichier de configuration (config.json → config.log).
func PathFor(cfgPath string) string {
	return strings.TrimSuffix(cfgPath, filepath.Ext(cfgPath)) + ".log"
}

// AppendInstall ajoute un bloc au fichier installation.log du dossier d'installation.
func AppendInstall(dir, text string) {
	if f, err := os.OpenFile(filepath.Join(dir, "installation.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		_, _ = f.WriteString(text)
		f.Close()
	}
}
