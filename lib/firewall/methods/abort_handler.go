package methods

import (
	"github.com/gofiber/fiber/v2"
)

func Abort(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}
