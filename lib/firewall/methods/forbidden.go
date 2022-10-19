package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Forbidden(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusForbidden)
	c.Abort()
}
