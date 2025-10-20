package firewall

import (
	"github.com/gofiber/fiber/v2"
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
	filters = []FilterInterface{
		&rules.SkipStaticFiles{},
		&rules.IpFilter{},
		&rules.DosDetector{},
		&rules.CookieCheckpoint{},
	}
}

func EnableRedis(enable bool) {
	cookieDb.EnableRedisClient(enable)
	countryDb.EnableRedisClient(enable)
	googleDb.EnableRedisClient(enable)
}

// executeFilters runs a slice of filters and handles the results
func executeFilters(c *fiber.Ctx, filters []FilterInterface, remoteIP, hostname string) error {
	var result FilterResult

	for _, filter := range filters {
		result = filter.Handler(c, remoteIP, hostname)

		if result.Passed {
			if result.BreakLoop { // stop filtering
				return c.Next()
			}
			continue // passed current filter, skip to next
		}

		// not passed current filter
		if result.Error != nil {
			log.Println("Error in firewall", result.Error.Error())
		}

		if result.AbortHandler != nil {
			return result.AbortHandler(c)
		}

		return methods.Forbidden(c)
	}

	return c.Next()
}

func Handler(c *fiber.Ctx) error {
	remoteIP := utils.ResolveRemoteIP(c)
	hostname := utils.ResolveHostname(c)

	return executeFilters(c, filters, remoteIP, hostname)
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
	botFilters = []FilterInterface{
		&custom.BlockSensitiveUrls{},
	}

	// Pre-allocate with exact capacity and convert to lowercase
	botUserAgents = make([]string, len(botUserAgentsRaw))
	for i, ua := range botUserAgentsRaw {
		botUserAgents[i] = strings.ToLower(ua)
	}
}

// isBot checks if the user agent matches known bot patterns
func isBot(userAgent string) bool {
	for _, botUA := range botUserAgents {
		if strings.Contains(userAgent, botUA) {
			return true
		}
	}
	return false
}

func BotHandler(c *fiber.Ctx) error {
	// Only process bot filters if this is a bot
	if !isBot(strings.ToLower(c.Get("User-Agent"))) {
		return c.Next()
	}

	remoteIP := utils.ResolveRemoteIP(c)
	hostname := utils.ResolveHostname(c)

	return executeFilters(c, botFilters, remoteIP, hostname)
}
