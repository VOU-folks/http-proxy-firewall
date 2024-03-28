package custom

import (
	"github.com/gin-gonic/gin"

	. "http-proxy-firewall/lib/firewall/interfaces"
	. "http-proxy-firewall/lib/firewall/rules"
)

type BlockSensitiveUrls struct {
}

func (bsu *BlockSensitiveUrls) Handler(c *gin.Context, remoteIP string, hostname string) FilterResult {
	if c.Request.URL.Query().Has("ps") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("pw") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("pwd") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("pass") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("password") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("secret") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("api_key") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("tkn") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("token") {
		return AbortRequestResult
	}
	if c.Request.URL.Query().Has("access_token") {
		return AbortRequestResult
	}

	return PassToNext
}
