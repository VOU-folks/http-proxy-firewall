package rules

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"

	. "http-proxy-firewall/lib/firewall/interfaces"
)

// staticFileExtensions contains common static file extensions that should skip firewall checks
var staticFileExtensions = []string{
	// Images
	".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg",
	".ico", ".bmp", ".tiff", ".avif",
	// Stylesheets
	".css",
	// Scripts
	".js",
	// Documents
	".htm", ".html", ".txt",
}

type SkipStaticFiles struct{}

func (ssf *SkipStaticFiles) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	ext := strings.ToLower(filepath.Ext(c.Path()))

	if slices.Contains(staticFileExtensions, ext) {
		return BreakLoopResult
	}

	return PassToNext
}
