package rules

import (
	"time"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/db/cookie"
	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/utils"
)

var cookieMaxAge = int((time.Hour * 24).Seconds())

type CookieCheckpoint struct {
}

var sidCookieName = "pf-sid"

var ServeNewSidResult = FilterResult{
	Error:        nil,
	AbortHandler: ServeNewSid,
	Passed:       false,
	BreakLoop:    true,
}

func (cc *CookieCheckpoint) Handler(c *gin.Context) FilterResult {
	remoteIP := utils.ResolveRemoteIP(c)

	sid, err := c.Cookie(sidCookieName)
	if err != nil {
		return ServeNewSidResult
	}

	valid := cookie.ValidateSid(
		sid,
		remoteIP,
		utils.ResolveHostname(c),
		c.Request.UserAgent(),
	)

	if !valid {
		return ServeNewSidResult
	}

	return PassToNext
}

func ServeNewSid(c *gin.Context) {
	remoteIP := utils.ResolveRemoteIP(c)
	hostname := utils.ResolveHostname(c)

	cookieRecord := cookie.NewCookieRecord(
		remoteIP,
		hostname,
		c.Request.UserAgent(),
	)

	cookie.StoreCookieRecord(cookieRecord)

	c.SetCookie(
		sidCookieName,
		cookieRecord.Sid,
		cookieMaxAge,
		"/",
		hostname,
		false,
		false,
	)

	c.Writer.WriteHeader(200)
	c.Writer.Header().Set("Content-Type", "text/html")
	c.Writer.Write([]byte("<meta http-equiv=\"refresh\" content=\"0\">"))
	c.Abort()
}
