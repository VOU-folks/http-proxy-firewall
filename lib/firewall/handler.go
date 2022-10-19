package firewall

import (
	"log"

	"github.com/gin-gonic/gin"

	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/firewall/methods"
	"http-proxy-firewall/lib/firewall/rules"
)

var filters = make([]FilterInterface, 0)

func init() {
	filters = append(filters, &rules.IpFilter{})
	filters = append(filters, &rules.CookieCheckpoint{})
}

func Handler(c *gin.Context) {
	var result FilterResult

	for _, filter := range filters {
		result = filter.Handler(c)

		if result.Passed {
			if result.BreakLoop { // stop filtering
				return
			}
			continue // passed current filter, skip to next
		}

		// not passed current filter
		if result.Error != nil {
			log.Println("Error in firewall", result.Error.Error())
		}

		if result.AbortHandler != nil {
			result.AbortHandler(c)
			return
		}

		methods.Forbidden(c)
	}
}
