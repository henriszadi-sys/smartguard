package wizard

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Certificat HTTPS auto-signé du module (option « Administration en HTTPS »
// de l'assistant, décochée par défaut). Il est créé dans le dossier « tls » du
// module et réutilisé tant qu'il est valide ; le client peut le remplacer par
// son propre certificat (champs tls_cert et tls_key de la configuration).

const (
	tlsDirName  = "tls"
	tlsCertName = "smartguard-cert.pem"
	tlsKeyName  = "smartguard-key.pem"
	tlsValidity = 10 * 365 * 24 * time.Hour
)

// tlsPaths renvoie les chemins du certificat et de la clé générés pour un module.
func tlsPaths(moduleDir string) (string, string) {
	d := filepath.Join(moduleDir, tlsDirName)
	return filepath.Join(d, tlsCertName), filepath.Join(d, tlsKeyName)
}

// isGeneratedTLS indique si le certificat configuré est celui généré par l'assistant.
func isGeneratedTLS(moduleDir, certPath string) bool {
	c, _ := tlsPaths(moduleDir)
	return certPath != "" && strings.EqualFold(filepath.Clean(certPath), filepath.Clean(c))
}

// ensureSelfSigned crée (ou réutilise s'il est encore valide) le certificat auto-signé du module.
func ensureSelfSigned(moduleDir string, hosts []string, now time.Time) (string, string, error) {
	certPath, keyPath := tlsPaths(moduleDir)
	if pair, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		if leaf, err := x509.ParseCertificate(pair.Certificate[0]); err == nil && now.Add(30*24*time.Hour).Before(leaf.NotAfter) {
			return certPath, keyPath, nil
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		return "", "", err
	}
	tpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "SmartGUARD", Organization: []string{"SmartGUARD (certificat auto-signé)"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(tlsValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if h = strings.TrimSpace(h); h == "" {
			continue
		}
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

// localHosts : noms et adresses du serveur à inscrire dans le certificat.
func localHosts() []string {
	hosts := []string{"localhost", "127.0.0.1", "::1"}
	if h, err := os.Hostname(); err == nil && h != "" {
		hosts = append(hosts, h, strings.ToLower(h))
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && !ipn.IP.IsLinkLocalUnicast() {
				hosts = append(hosts, ipn.IP.String())
			}
		}
	}
	return hosts
}

// Scheme renvoie « https » si le module est configuré en HTTPS, « http » sinon.
func Scheme(tlsCert, tlsKey string) string {
	if tlsCert != "" && tlsKey != "" {
		return "https"
	}
	return "http"
}
