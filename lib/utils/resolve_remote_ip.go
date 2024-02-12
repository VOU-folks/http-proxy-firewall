package utils

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func ResolveRemoteIP(c *gin.Context) string {
	cfIP := strings.TrimSpace(c.Request.Header.Get("CF-Connecting-IP"))
	if cfIP != "" {
		return cfIP
	}
	return c.ClientIP()
}
