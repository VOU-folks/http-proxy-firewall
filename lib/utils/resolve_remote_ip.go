package utils

import (
	"github.com/gin-gonic/gin"
)

func ResolveRemoteIP(c *gin.Context) string {
	return c.ClientIP()
}
