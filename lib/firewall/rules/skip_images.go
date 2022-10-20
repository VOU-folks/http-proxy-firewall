package rules

import (
	"mime"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	. "http-proxy-firewall/lib/firewall/interfaces"
)

type SkipImages struct {
}

func isImageUrl(url *url.URL) bool {
	ext := filepath.Ext(url.Path)
	mime := mime.TypeByExtension(ext)
	return strings.Contains(mime, "image/")
}

func (si *SkipImages) Handler(c *gin.Context) FilterResult {
	if isImageUrl(c.Request.URL) {
		return BreakLoopResult
	}

	return PassToNext
}
