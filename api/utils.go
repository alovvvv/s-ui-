package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/alireza0/s-ui/logger"

	"github.com/gin-gonic/gin"
)

type Msg struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Obj     interface{} `json:"obj"`
}

// getRemoteIp returns the client address, via gin's trusted-proxy handling.
// Reading X-Forwarded-For directly meant any client could put anything in the
// login log and in the key the rate limiter counts against.
func getRemoteIp(c *gin.Context) string {
	return c.ClientIP()
}

func getHostname(c *gin.Context) string {
	host := c.Request.Host
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(c.Request.Host)
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
	}
	return host
}

func jsonMsg(c *gin.Context, msg string, err error) {
	jsonMsgObj(c, msg, nil, err)
}

func jsonObj(c *gin.Context, obj interface{}, err error) {
	jsonMsgObj(c, "", obj, err)
}

func jsonMsgObj(c *gin.Context, msg string, obj interface{}, err error) {
	m := Msg{
		Obj: obj,
	}
	if err == nil {
		m.Success = true
		if msg != "" {
			m.Msg = msg
		}
	} else {
		m.Success = false
		m.Msg = msg + ": " + err.Error()
		logger.Warning("failed :", err)
	}
	c.JSON(http.StatusOK, m)
}

func pureJsonMsg(c *gin.Context, success bool, msg string) {
	if success {
		c.JSON(http.StatusOK, Msg{
			Success: true,
			Msg:     msg,
		})
	} else {
		c.JSON(http.StatusOK, Msg{
			Success: false,
			Msg:     msg,
		})
	}
}

func checkLogin(c *gin.Context) {
	if IsLogin(c) {
		c.Next()
		return
	}
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		// 401, not 200. The body still carries the old message so an older
		// frontend keeps working, but a client should not have to match on
		// English prose to find out its session expired.
		c.JSON(http.StatusUnauthorized, Msg{Success: false, Msg: "Invalid login"})
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
	c.Abort()
}
