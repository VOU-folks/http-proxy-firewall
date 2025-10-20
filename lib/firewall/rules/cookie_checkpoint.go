package rules

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"http-proxy-firewall/lib/db/cookie"
	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/utils"
)

var cookieMaxAge = int((time.Hour * 24).Seconds())

type CookieCheckpoint struct {
}

var sidCookieName = "_X-SID_"

var ServeNewSidResult = FilterResult{
	Error:        nil,
	AbortHandler: ServeNewSid,
	Passed:       false,
	BreakLoop:    true,
}

func (cc *CookieCheckpoint) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	sid := c.Cookies(sidCookieName)
	if sid == "" {
		return ServeNewSidResult
	}

	valid := cookie.ValidateSid(
		sid,
		remoteIP,
		hostname,
		c.Get("User-Agent"),
	)

	if !valid {
		return ServeNewSidResult
	}

	return PassToNext
}

func ServeNewSid(c *fiber.Ctx) error {
	remoteIP := utils.ResolveRemoteIP(c)
	hostname := utils.ResolveHostname(c)

	cookieRecord := cookie.NewCookieRecord(
		remoteIP,
		hostname,
		c.Get("User-Agent"),
	)

	cookie.StoreCookieRecord(cookieRecord)

	c.Cookie(&fiber.Cookie{
		Name:     sidCookieName,
		Value:    cookieRecord.Sid,
		MaxAge:   cookieMaxAge,
		Path:     "/",
		Domain:   hostname,
		Secure:   false,
		HTTPOnly: false,
	})

	c.Set("Content-Type", "text/html")
	return c.SendString("<meta http-equiv=\"refresh\" content=\"0\">")
}
