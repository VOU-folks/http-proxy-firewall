package firewall

import (
	"log"

	"github.com/gin-gonic/gin"

	cookieDb "http-proxy-firewall/lib/db/cookie"
	countryDb "http-proxy-firewall/lib/db/country"
	googleDb "http-proxy-firewall/lib/db/google"
	"http-proxy-firewall/lib/utils"

	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/firewall/methods"
	"http-proxy-firewall/lib/firewall/rules"
)

var filters []FilterInterface

func init() {
	filters = make([]FilterInterface, 0)
	filters = append(filters, &rules.SkipStaticFiles{})
	filters = append(filters, &rules.IpFilter{})
	filters = append(filters, &rules.DosDetector{})
	filters = append(filters, &rules.CookieCheckpoint{})
}

func EnableRedis(enable bool) {
	cookieDb.EnableRedisClient(enable)
	countryDb.EnableRedisClient(enable)
	googleDb.EnableRedisClient(enable)
}

func Handler(c *gin.Context) {
	var result FilterResult

	remoteIP := utils.ResolveRemoteIP(c)
	hostname := utils.ResolveHostname(c)

	for _, filter := range filters {
		result = filter.Handler(c, remoteIP, hostname)

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
