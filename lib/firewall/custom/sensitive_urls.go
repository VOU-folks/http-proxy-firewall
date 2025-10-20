package custom

import (
	"github.com/gofiber/fiber/v2"

	. "http-proxy-firewall/lib/firewall/interfaces"
	. "http-proxy-firewall/lib/firewall/rules"
)

// sensitiveParams is a list of query parameter names that should not be indexed by bots
var sensitiveParams = []string{
	"ps", "pw", "pwd", "pass", "password",
	"secret", "api_key", "tkn", "token", "access_token",
}

type BlockSensitiveUrls struct {
}

func (bsu *BlockSensitiveUrls) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	// Get all query parameters at once (single parse)
	queries := c.Queries()

	// Check if any sensitive parameter exists in the query
	for _, sensitive := range sensitiveParams {
		if _, exists := queries[sensitive]; exists {
			return AbortRequestResult
		}
	}

	return PassToNext
}
