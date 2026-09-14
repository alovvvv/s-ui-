package api

import (
	"encoding/gob"
	"net"
	"net/http"
	"strings"

	"github.com/alireza0/s-ui/database/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginUser = "LOGIN_USER"
)

func init() {
	gob.Register(model.User{})
}

// BaseSessionOptions are the cookie attributes that do not depend on the
// request, so the session store and every login share one definition.
//
// HttpOnly keeps the session out of reach of script: the panel renders
// operator-supplied strings in several places, and without it any one of them
// turning into an XSS hands over the session outright. The frontend router
// used to read this cookie to decide whether it was logged in, which is why
// this could not be set before; it now tracks that itself.
//
// SameSite=Strict is safe in both HTTP and HTTPS mode -- it has nothing to do
// with TLS -- and stops a cross-site request from carrying the session at all.
func BaseSessionOptions(maxAgeMinutes int) sessions.Options {
	o := sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	if maxAgeMinutes > 0 {
		o.MaxAge = maxAgeMinutes * 60
	}
	return o
}

func sessionOptions(c *gin.Context, maxAgeMinutes int) sessions.Options {
	o := BaseSessionOptions(maxAgeMinutes)
	o.Secure = requestIsHTTPS(c)
	return o
}

// requestIsHTTPS reports whether the browser reached the panel over TLS.
//
// It must be derived per request, never hardcoded: a browser will not send a
// Secure cookie over plain HTTP, so a fixed true breaks login on every
// HTTP-only install, and a fixed false gives up the protection on HTTPS ones.
//
// Behind a reverse proxy the panel itself speaks plain HTTP, so the forwarded
// scheme is the only evidence there. It is believed only from a loopback or
// private peer -- otherwise anyone able to reach an HTTP-only panel directly
// could set the header and make the browser refuse to send the cookie back,
// locking the operator out of their own panel.
func requestIsHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if !strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		return false
	}
	ip := net.ParseIP(c.RemoteIP())
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func SetLoginUser(c *gin.Context, userName string, maxAge int) error {
	s := sessions.Default(c)
	s.Set(loginUser, userName)
	s.Options(sessionOptions(c, maxAge))
	return s.Save()
}

func SetMaxAge(c *gin.Context) error {
	s := sessions.Default(c)
	s.Options(sessionOptions(c, 0))
	return s.Save()
}

func GetLoginUser(c *gin.Context) string {
	s := sessions.Default(c)
	obj := s.Get(loginUser)
	if obj == nil {
		return ""
	}
	objStr, ok := obj.(string)
	if !ok {
		return ""
	}
	return objStr
}

func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != ""
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	// Same attributes as the cookie being replaced. A browser matches the
	// deletion against Path and Secure, so an expiry written with different
	// attributes leaves the original cookie in place.
	o := sessionOptions(c, 0)
	o.MaxAge = -1
	s.Options(o)
	s.Save()
}
