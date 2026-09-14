package api

import (
	cryptotls "crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func ctxFor(t *testing.T, tls bool, remoteAddr string, headers map[string]string) *gin.Context {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if tls {
		req.TLS = &cryptotls.ConnectionState{}
	} else {
		req.TLS = nil
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c
}

// Secure has to follow the connection. A hardcoded true stops a browser sending
// the cookie at all on an HTTP-only install, which breaks login outright; a
// hardcoded false gives the protection away on an HTTPS one.
func TestRequestIsHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		tls        bool
		remoteAddr string
		headers    map[string]string
		want       bool
	}{
		{"plain http", false, "203.0.113.9:5000", nil, false},
		{"direct tls", true, "203.0.113.9:5000", nil, true},

		// A reverse proxy on the same host or the LAN is the only place the
		// forwarded scheme is worth anything.
		{"proxy on loopback", false, "127.0.0.1:5000",
			map[string]string{"X-Forwarded-Proto": "https"}, true},
		{"proxy on a private address", false, "10.1.2.3:5000",
			map[string]string{"X-Forwarded-Proto": "https"}, true},
		{"proxy header casing", false, "127.0.0.1:5000",
			map[string]string{"X-Forwarded-Proto": "HTTPS"}, true},

		// From anywhere else the header is the client's own claim. Believing it
		// would let anyone who can reach an HTTP-only panel make the browser
		// stop sending the session cookie back.
		{"forged from the internet", false, "203.0.113.9:5000",
			map[string]string{"X-Forwarded-Proto": "https"}, false},
		{"proxy reporting http", false, "127.0.0.1:5000",
			map[string]string{"X-Forwarded-Proto": "http"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := ctxFor(t, tc.tls, tc.remoteAddr, tc.headers)
			if got := requestIsHTTPS(c); got != tc.want {
				t.Errorf("requestIsHTTPS = %v, want %v", got, tc.want)
			}
		})
	}
}

// The attributes that do not depend on the connection must be set whatever the
// mode, and the session lifetime has to reach the cookie.
func TestBaseSessionOptions(t *testing.T) {
	o := BaseSessionOptions(0)
	if !o.HttpOnly {
		t.Error("HttpOnly is not set: an XSS anywhere in the panel would hand over the session")
	}
	if o.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", o.SameSite)
	}
	if o.Path != "/" {
		t.Errorf("Path = %q, want /", o.Path)
	}
	if o.MaxAge != 0 {
		t.Errorf("MaxAge = %d, want 0 for an unset session age", o.MaxAge)
	}
	if got := BaseSessionOptions(90).MaxAge; got != 90*60 {
		t.Errorf("MaxAge = %d, want %d seconds", got, 90*60)
	}
}

// Secure is never baked into the base options; it is the per-request half.
func TestSessionOptionsDerivesSecure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if BaseSessionOptions(0).Secure {
		t.Error("the base options set Secure, which would break HTTP mode")
	}
	if sessionOptions(ctxFor(t, false, "203.0.113.9:5000", nil), 0).Secure {
		t.Error("Secure was set on a plain HTTP request")
	}
	if !sessionOptions(ctxFor(t, true, "203.0.113.9:5000", nil), 0).Secure {
		t.Error("Secure was not set on a TLS request")
	}
}
