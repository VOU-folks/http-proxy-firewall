package rules

import (
	"time"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/db/cookie"
	. "http-proxy-firewall/lib/firewall/interfaces"
)

var cookieMaxAge = int((time.Hour * 24).Seconds())

type CookieCheckpoint struct {
}

var sidCookieName = "X-SID"

var ServeNewSidResult = FilterResult{
	Error:        nil,
	AbortHandler: ServeNewSid,
	Passed:       false,
	BreakLoop:    true,
}

func (cc *CookieCheckpoint) Handler(c *gin.Context) FilterResult {
	remoteIP := c.RemoteIP()

	sid, err := c.Cookie(sidCookieName)
	if err != nil {
		return ServeNewSidResult
	}

	valid := cookie.ValidateSid(
		sid,
		remoteIP,
		c.Request.Host,
		c.Request.UserAgent(),
	)
	if !valid {
		return ServeNewSidResult
	}

	return PassToNext
}

func ServeNewSid(c *gin.Context) {
	remoteIP := c.RemoteIP()

	cookieRecord := cookie.NewCookieRecord(
		remoteIP,
		c.Request.Host,
		c.Request.UserAgent(),
	)

	cookie.StoreCookieRecord(cookieRecord)

	c.SetCookie(
		sidCookieName,
		cookieRecord.Sid,
		cookieMaxAge,
		"/",
		c.Request.Host,
		false,
		false,
	)

	c.Writer.WriteHeader(200)
	c.Writer.Header().Set("Content-Type", "text/html")
	c.Writer.Write([]byte("<meta http-equiv=\"refresh\" content=\"0\">"))
	c.Abort()
}
