package rules

import (
	"net"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/db/country"
	"http-proxy-firewall/lib/db/google"
	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/firewall/methods"
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
}

type IpFilter struct {
}

var ipWhitelist = []string{
	"185.22.155.62",
	"185.22.155.63",
	"31.31.198.237",
	"37.187.132.146",
	"37.187.132.13",
	"167.233.11.197",

	// internal ips
	"162.55.45.158",
	"49.12.112.9",
	"10.0.0.1",
	"10.0.0.2",
	"10.0.0.3",
	"10.0.0.4",
}

func isIpWhitelisted(ipAddress string) bool {
	for _, ip := range ipWhitelist {
		if ip == ipAddress {
			return true
		}
	}
	return false
}

var allowedCountries = []string{
	"Azerbaijan",
	"Turkey",
	"Ukraine",
	"Georgia",
	"Russia",
	"Portugal",
}

func isCountryAllowed(country string) bool {
	return slices.Contains(allowedCountries, country)
}

var breakLoopResult = FilterResult{
	Error:     nil,
	Passed:    true,
	BreakLoop: true,
}

var passToNext = FilterResult{
	Error:     nil,
	Passed:    true,
	BreakLoop: false,
}

var abortRequestResult = FilterResult{
	Error:        nil,
	Passed:       false,
	BreakLoop:    false,
	AbortHandler: methods.Forbidden,
}

func (f IpFilter) Handler(c *gin.Context) FilterResult {
	remoteIP := c.RemoteIP()
	ip := net.ParseIP(remoteIP)

	breakLoop := loopbackV4.Contains(ip) || loopbackV6.Contains(ip) ||
		isIpWhitelisted(remoteIP) || google.IsGoogleBot(ip)
	if breakLoop {
		return breakLoopResult
	}

	resolvedCountry := country.ResolveCountryByIP(remoteIP)
	if resolvedCountry != "" {
		if isCountryAllowed(resolvedCountry) {
			return breakLoopResult
		}

		result := abortRequestResult
		result.AbortHandler = methods.ForbiddenCountry(resolvedCountry, remoteIP)

		return result
	}

	// cannot detect country
	// pass to next filter
	return passToNext
}
