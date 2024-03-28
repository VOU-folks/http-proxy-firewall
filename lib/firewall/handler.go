package firewall

import (
	"github.com/gin-gonic/gin"
	cookieDb "http-proxy-firewall/lib/db/cookie"
	countryDb "http-proxy-firewall/lib/db/country"
	googleDb "http-proxy-firewall/lib/db/google"
	"http-proxy-firewall/lib/utils"
	"log"
	"strings"

	"http-proxy-firewall/lib/firewall/custom"
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

var botUserAgents []string
var botUserAgentsRaw = []string{
	"Googlebot",
	"Googlebot-Image",
	"Googlebot-News",
	"Googlebot-Video",
	"AdsBot-Google",
	"Mediapartners-Google",
	"APIs-Google",
	"FeedFetcher-Google",
	"AppEngine-Google",
	"Google-Read-Aloud",
	"Google-SearchByImage",
	"Google-SearchByVoice",
	"Google-Favicon",
	"Google-SearchConsole",
	"Google-StructuredDataTestingTool",
	"Google-Adwords",
	"AhrefsBot",
	"Bingbot",
	"YandexBot",
	"YandexImages",
	"YandexVideo",
	"YandexMedia",
	"YandexBlogs",
	"YandexFavicons",
	"YandexWebmaster",
	"YandexPagechecker",
	"YandexImageResizer",
	"YandexDirect",
	"YandexAdNet",
	"YandexDirectDyn",
	"YandexMarket",
	"YandexVertis",
	"YandexCalendar",
	"YandexSitelinks",
	"YandexMetrika",
	"YandexNews",
	"YandexCatalog",
	"YandexAntivirus",
	"YandexMarket",
	"YandexFlights",
	"Amazonbot",
	"Slurp",
	"msnbot",
	"bingbot",
	"bingpreview",
	"adidxbot",
}
var botFilters []FilterInterface

func init() {
	botFilters = make([]FilterInterface, 0)
	botFilters = append(botFilters, &custom.BlockSensitiveUrls{})

	botUserAgents = make([]string, 0)
	for _, ua := range botUserAgentsRaw {
		botUserAgents = append(botUserAgents, strings.ToLower(ua))
	}
}

func BotHandler(c *gin.Context) {
	var result FilterResult

	remoteIP := utils.ResolveRemoteIP(c)
	hostname := utils.ResolveHostname(c)

	userAgent := strings.ToLower(c.Request.UserAgent())

	isBot := false

	for _, botUA := range botUserAgents {
		if strings.Contains(userAgent, botUA) {
			isBot = true
			break
		}
	}

	if isBot {
		for _, filter := range botFilters {
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

			//log.Println("Bot blocked", c.Request.Host, c.Request.URL.String())

			if result.AbortHandler != nil {
				result.AbortHandler(c)
				return
			}

			methods.Forbidden(c)
		}
	}
}
