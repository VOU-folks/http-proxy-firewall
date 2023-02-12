package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func ResolveHostname(c *gin.Context) string {
	host := c.Request.Host
	if strings.HasPrefix(host, "www.") {
		host = strings.TrimLeft(host, "www.")
	}

	if strings.Contains(host, ":") {
		splitted := strings.Split(host, ":")
		host = splitted[0]
	}

	return host
}
