package methods

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Abort(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusNoContent)
	c.Abort()
}
