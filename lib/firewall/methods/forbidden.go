package methods

import (
	"github.com/gofiber/fiber/v2"
)

func Forbidden(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusForbidden)
}
