package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// SameOrigin rejects state-changing requests that did not come from the panel's
// own pages.
//
// The session is a cookie, so without this any page the operator visits while
// logged in can post to the panel in their name -- adding a client, changing
// the password, importing a database. SameSite=Strict on the cookie already
// covers current browsers; this is the half that does not depend on the
// browser being current.
//
// Only the host is compared, never a hardcoded scheme. The panel has to work
// on plain HTTP as well as HTTPS, and behind a TLS-terminating proxy the
// scheme the browser used and the one the panel sees differ anyway, so
// requiring https:// here would reject every legitimate request in two of the
// three deployments.
func SameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = c.GetHeader("Referer")
		}
		if origin == "" {
			// Neither header. A cross-site form post cannot set a custom
			// header, so the panel's own XHR marker is enough to tell the two
			// apart -- and it is already what checkLogin answers on.
			if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || !strings.EqualFold(u.Host, c.Request.Host) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
