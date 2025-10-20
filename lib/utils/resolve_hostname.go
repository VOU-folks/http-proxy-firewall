package utils

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ResolveHostname(c *fiber.Ctx) string {
	// Try X-Forwarded-Host first
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Hostname()
	}

	// Remove www. prefix
	host = strings.TrimPrefix(host, "www.")

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	return host
}
