package rules

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"http-proxy-firewall/lib/db/cookie"
	. "http-proxy-firewall/lib/firewall/interfaces"
)

const (
	sidCookieName        = "_X-SID_"
	htmlAutoRefresh      = "<meta http-equiv=\"refresh\" content=\"0\">"
	headerContentType    = "Content-Type"
	headerContentTypeVal = "text/html"
)

var cookieMaxAge = int((time.Hour * 24).Seconds())

type CookieCheckpoint struct {
}

func (cc *CookieCheckpoint) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	sid := c.Cookies(sidCookieName)
	if sid == "" {
		return createServeNewSidResult(remoteIP, hostname, c.Get("User-Agent"))
	}

	// Cache User-Agent to avoid duplicate c.Get() call
	userAgent := c.Get("User-Agent")
	valid := cookie.ValidateSid(sid, remoteIP, hostname, userAgent)

	if !valid {
		return createServeNewSidResult(remoteIP, hostname, userAgent)
	}

	return PassToNext
}

// createServeNewSidResult creates a FilterResult with a closure that captures the context
func createServeNewSidResult(remoteIP, hostname, userAgent string) FilterResult {
	return FilterResult{
		Error:     nil,
		Passed:    false,
		BreakLoop: true,
		AbortHandler: func(c *fiber.Ctx) error {
			return serveNewSid(c, remoteIP, hostname, userAgent)
		},
	}
}

// serveNewSid creates a new session cookie and returns an auto-refresh page
func serveNewSid(c *fiber.Ctx, remoteIP, hostname, userAgent string) error {
	cookieRecord := cookie.NewCookieRecord(remoteIP, hostname, userAgent)
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

	c.Set(headerContentType, headerContentTypeVal)
	return c.SendString(htmlAutoRefresh)
}
