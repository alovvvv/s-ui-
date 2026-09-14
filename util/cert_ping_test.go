package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// tlsServerWith starts a TLS listener presenting one self-signed certificate,
// with exactly the subject alternative names given.
func tlsServerWith(t *testing.T, dnsNames []string) (host, port string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dnsNames,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				if tc, ok := conn.(*tls.Conn); ok {
					_ = tc.Handshake()
				}
				time.Sleep(50 * time.Millisecond)
				conn.Close()
			}()
		}
	}()

	host, port, err = net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

// The loop picks the first certificate carrying a SAN and leaves leaf nil when
// none does. Reading through it took the whole panel down, from a form field
// an admin types a hostname into.
func TestGetTlsPingWithoutSubjectAltNames(t *testing.T) {
	host, port := tlsServerWith(t, nil)

	_, err := GetTlsPing(host, port)
	if err == nil {
		t.Fatal("a certificate with no SAN was accepted; expected an error")
	}
}

func TestGetTlsPingReturnsTheLeafHash(t *testing.T) {
	host, port := tlsServerWith(t, []string{"example.test"})

	got, err := GetTlsPing(host, port)
	if err != nil {
		t.Fatalf("GetTlsPing: %v", err)
	}
	m, ok := got.(map[string]string)
	if !ok {
		t.Fatalf("result is %T, want map[string]string", got)
	}
	if m["leafHash"] == "" {
		t.Error("leafHash is empty")
	}
}

func TestGetTlsPingRejectsAnEmptyDomain(t *testing.T) {
	if _, err := GetTlsPing("", "443"); err == nil {
		t.Error("an empty domain was accepted")
	}
}
