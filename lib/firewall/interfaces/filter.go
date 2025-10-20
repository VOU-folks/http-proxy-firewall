package interfaces

import (
	"github.com/gofiber/fiber/v2"
)

type FilterInterface interface {
	Handler(c *fiber.Ctx, ip string, hostname string) FilterResult
}

type FilterResult struct {
	Error        error
	AbortHandler func(c *fiber.Ctx) error
	Passed       bool
	BreakLoop    bool
}
