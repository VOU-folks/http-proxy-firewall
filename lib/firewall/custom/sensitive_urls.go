package custom

import (
	"github.com/gofiber/fiber/v2"

	. "http-proxy-firewall/lib/firewall/interfaces"
	. "http-proxy-firewall/lib/firewall/rules"
)

type BlockSensitiveUrls struct {
}

func (bsu *BlockSensitiveUrls) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	sensitiveParams := []string{
		"ps", "pw", "pwd", "pass", "password",
		"secret", "api_key", "tkn", "token", "access_token",
	}

	for _, param := range sensitiveParams {
		if c.Query(param) != "" {
			return AbortRequestResult
		}
	}

	return PassToNext
}
