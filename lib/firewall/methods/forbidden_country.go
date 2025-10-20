package methods

import (
	"github.com/gofiber/fiber/v2"
)

func ForbiddenCountry(country string, ip string) func(ctx *fiber.Ctx) error {
	// Don't leak country/IP information to client for security
	return func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusForbidden)
	}
}
