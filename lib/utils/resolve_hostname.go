package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func ResolveHostname(c *gin.Context) string {
	host := c.Request.Host
	if strings.HasPrefix(host, "www.") {
		host = strings.TrimLeft(host, "www.")
	}
	host, _, _ = net.SplitHostPort(host)

	return host
}
