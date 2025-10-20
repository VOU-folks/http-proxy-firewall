package utils

import (
	"github.com/gofiber/fiber/v2"
	"strings"
)

func ResolveRemoteIP(c *fiber.Ctx) string {
	cfIP := strings.TrimSpace(c.Get("CF-Connecting-IP"))
	if cfIP != "" {
		return cfIP
	}

	// Try X-Forwarded-For header
	xff := strings.TrimSpace(c.Get("X-Forwarded-For"))
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fallback to direct IP
	return c.IP()
}
