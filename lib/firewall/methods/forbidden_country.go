package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ForbiddenCountry(country string, ip string) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Write([]byte("Forbidden country: " + country + " [ip: " + ip + "]"))
		c.Abort()
	}
}
