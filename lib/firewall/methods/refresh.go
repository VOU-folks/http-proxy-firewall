package methods

import (
	"github.com/gofiber/fiber/v2"
)

func Refresh(c *fiber.Ctx) error {
	return c.Redirect(c.OriginalURL(), fiber.StatusFound)
}
