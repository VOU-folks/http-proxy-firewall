package methods

import (
	"github.com/gofiber/fiber/v2"
)

func ForbiddenCountry(country string, ip string) func(ctx *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden country: " + country + " [ip: " + ip + "]")
	}
}
