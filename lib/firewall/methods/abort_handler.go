package methods

import (
	"github.com/gin-gonic/gin"
)

func Abort(c *gin.Context) {
	c.Writer.WriteHeader(204)
	c.Abort()
}
