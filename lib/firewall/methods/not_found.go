package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NotFound(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusNotFound)
	c.Abort()
}
