package rules

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/db/country"
	"http-proxy-firewall/lib/db/google"
	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/utils"
	"http-proxy-firewall/lib/utils/slices"
)

var loopbackV4 *net.IPNet
var loopbackV6 *net.IPNet

func init() {
	var network *net.IPNet
	_, network, _ = net.ParseCIDR("127.0.0.1/8")
	loopbackV4 = network

	_, network, _ = net.ParseCIDR("127.0.0.1/8")
	loopbackV6 = network

	whitelist := strings.Split(utils.GetEnv("IP_FILTER_WHITELIST"), ",")
	for _, elem := range whitelist {
		ipWhitelist = append(ipWhitelist, elem)
	}

	countries := strings.Split(utils.GetEnv("IP_FILTER_ALLOWED_COUNTRIES"), ",")
	for _, elem := range countries {
		allowedCountries = append(allowedCountries, strings.Trim(elem, " "))
	}

	fmt.Println(ipWhitelist)
	fmt.Println(allowedCountries)
}

type IpFilter struct {
}

var ipWhitelist []string

func isIpWhitelisted(ipAddress string) bool {
	for _, ip := range ipWhitelist {
		if ip == ipAddress {
			return true
		}
	}
	return false
}

var allowedCountries []string

func isCountryAllowed(country string) bool {
	return slices.Contains(allowedCountries, country)
}

func (f *IpFilter) Handler(c *gin.Context) FilterResult {
	remoteIP := utils.ResolveRemoteIP(c)
	ip := net.ParseIP(remoteIP)

	breakLoop := loopbackV4.Contains(ip) || loopbackV6.Contains(ip) ||
		isIpWhitelisted(remoteIP) || google.IsGoogleBot(ip)
	if breakLoop {
		return BreakLoopResult
	}

	resolvedCountry := country.ResolveCountryByIP(remoteIP)
	if resolvedCountry != "" {
		if isCountryAllowed(resolvedCountry) {
			return BreakLoopResult
		}

		// result := AbortRequestResult
		// result.AbortHandler = methods.ForbiddenCountry(resolvedCountry, remoteIP)
		// return result

		return PassToNext
	}

	// cannot detect country
	// pass to next filter
	return PassToNext
}
