package rules

import (
	"mime"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	. "http-proxy-firewall/lib/firewall/interfaces"
)

type SkipStaticFiles struct {
}

func isImage(mime string) bool {
	return strings.Contains(mime, "image/")
}

func isCSS(ext string) bool {
	return ext == ".css"
}

func isJS(ext string) bool {
	return ext == ".js"
}

func isPlainTextFile(ext string) bool {
	return ext == ".htm" || ext == ".html" || ext == ".txt"
}

func (ssf *SkipStaticFiles) Handler(c *gin.Context) FilterResult {
	ext := strings.ToLower(filepath.Ext(c.Request.URL.Path))
	mime := mime.TypeByExtension(ext)

	if isImage(mime) ||
		isCSS(ext) || isJS(ext) ||
		isPlainTextFile(ext) {
		return BreakLoopResult
	}

	return PassToNext
}
