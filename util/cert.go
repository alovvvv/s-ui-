package util

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"net"
	"os"
	"strings"
	"time"

	"github.com/alireza0/s-ui/util/common"
	utls "github.com/refraction-networking/utls"
)

func CertPEMFromTLS(tlsConfig map[string]interface{}) string {
	if tlsConfig == nil {
		return ""
	}
	switch c := tlsConfig["certificate"].(type) {
	case string:
		if c != "" {
			return c
		}
	case []interface{}:
		lines := make([]string, 0, len(c))
		for _, l := range c {
			if s, ok := l.(string); ok {
				lines = append(lines, s)
			}
		}
		if len(lines) > 0 {
			return strings.Join(lines, "\n")
		}
	}
	if path, ok := tlsConfig["certificate_path"].(string); ok && path != "" {
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return ""
}

func parseLeafCert(pemData string) *x509.Certificate {
	rest := []byte(pemData)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil
			}
			return cert
		}
	}
}

// CertIsSelfSigned reports whether the leaf certificate in pemData is
// self-signed, i.e. its signature verifies against its own public key. Only
// self-signed certificates should be pinned via certificate_public_key_sha256;
// CA-signed certificates are validated normally.
func CertIsSelfSigned(pemData string) bool {
	cert := parseLeafCert(pemData)
	if cert == nil {
		return false
	}
	return cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) == nil
}

// CertPublicKeySha256 returns the base64-encoded SHA256 of the certificate's
// SubjectPublicKeyInfo (sing-box `certificate_public_key_sha256` / link pinSHA256).
func CertPublicKeySha256(pemData string) string {
	cert := parseLeafCert(pemData)
	if cert == nil {
		return ""
	}
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// CertSha256Hex returns the lowercase hex SHA256 of the whole certificate (DER),
// matching `openssl x509 -fingerprint -sha256` and Clash/mihomo's `fingerprint`.
func CertSha256Hex(pemData string) string {
	cert := parseLeafCert(pemData)
	if cert == nil {
		return ""
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

func GetTlsPing(domain string, port string) (any, error) {
	if domain == "" {
		return "", common.NewError("domain is empty")
	}
	if port == "" {
		port = "443"
	}

	d := net.Dialer{Timeout: 10 * time.Second}
	tcpConn, err := d.Dial("tcp", net.JoinHostPort(domain, port))
	if err != nil {
		return "", common.NewErrorf("Failed to dial tcp: %s", err)
	}
	// Closed on every path. This is reachable from an admin form, so a domain
	// that handshakes but serves no usable certificate used to leak a socket
	// per attempt.
	defer tcpConn.Close()

	tlsConn := utls.UClient(tcpConn, &utls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2", "http/1.1"},
	}, utls.HelloChrome_Auto)
	err = tlsConn.Handshake()
	if err != nil {
		return "", common.NewErrorf("Failed to handshake: %s", err)
	}
	var leaf *x509.Certificate
	for _, cert := range tlsConn.ConnectionState().PeerCertificates {
		if len(cert.DNSNames) != 0 {
			leaf = cert
			break
		}
	}
	// A server that presents only certificates without a SAN leaves leaf nil,
	// and reading through it took the whole panel down from an admin form.
	if leaf == nil {
		return "", common.NewErrorf("%s presented no certificate with a subject alternative name", domain)
	}
	sum := sha256.Sum256(leaf.RawSubjectPublicKeyInfo)
	leafObj := map[string]string{
		"leafHash": base64.StdEncoding.EncodeToString(sum[:]),
	}

	return leafObj, nil

}
