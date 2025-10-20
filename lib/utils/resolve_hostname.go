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

	if strings.HasPrefix(host, "www.") {
		host = strings.TrimLeft(host, "www.")
	}

	if strings.Contains(host, ":") {
		splitted := strings.Split(host, ":")
		host = splitted[0]
	}

	return host
}
