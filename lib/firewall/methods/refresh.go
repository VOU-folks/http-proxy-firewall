package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Refresh(c *gin.Context) {
	c.Redirect(http.StatusFound, c.Request.URL.String())
}
