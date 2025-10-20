package rules

import (
	"log"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"

	"http-proxy-firewall/lib/db/country"
	"http-proxy-firewall/lib/db/google"
	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/firewall/methods"
	"http-proxy-firewall/lib/utils"
	"http-proxy-firewall/lib/utils/slices"
)

var whitelistNetworks []*net.IPNet

func init() {
	var network *net.IPNet
	whitelistNetworks = make([]*net.IPNet, 0, 10)

	_, network, _ = net.ParseCIDR("127.0.0.1/8")
	whitelistNetworks = append(whitelistNetworks, network)

	envWhitelistNets := utils.GetEnv("IP_FILTER_WHITELIST_NETWORKS")
	whitelistNets := strings.Split(envWhitelistNets, ",")
	for _, elem := range whitelistNets {
		_, network, _ = net.ParseCIDR(elem)
		whitelistNetworks = append(whitelistNetworks, network)
	}

	envWhitelist := utils.GetEnv("IP_FILTER_WHITELIST")
	whitelist := strings.Split(envWhitelist, ",")
	for _, elem := range whitelist {
		ipWhitelist = append(ipWhitelist, elem)
	}

	envCountries := utils.GetEnv("IP_FILTER_ALLOWED_COUNTRIES")
	countries := strings.Split(envCountries, ",")
	for _, elem := range countries {
		allowedCountries = append(allowedCountries, strings.Trim(elem, " "))
	}

	envBlacklistedCountries := utils.GetEnv("IP_FILTER_BLACKLISTED_COUNTRIES")
	blacklistedCountriesSlice := strings.Split(envBlacklistedCountries, ",")
	for _, elem := range blacklistedCountriesSlice {
		trimmed := strings.Trim(elem, " ")
		if trimmed != "" {
			blacklistedCountries = append(blacklistedCountries, trimmed)
		}
	}

	log.Println("ip whitelist =", envWhitelist)
	log.Println("country whitelist =", envCountries)
	log.Println("country blacklist =", envBlacklistedCountries)
}

type IpFilter struct {
}

var ipWhitelist []string

func isIpInWhitelistedNetwork(ip net.IP) bool {
	for _, network := range whitelistNetworks {
		if network.Contains(ip) {
			//log.Println("IP", ip, "is in whitelisted network", network.String())
			return true
		}
	}
	return false
}

func isIpWhitelisted(ipAddress string) bool {
	for _, ip := range ipWhitelist {
		if ip == ipAddress {
			return true
		}
	}
	return false
}

var allowedCountries []string
var blacklistedCountries []string

func isCountryAllowed(country string) bool {
	return slices.Contains(allowedCountries, country)
}

func isCountryBlacklisted(country string) bool {
	return slices.Contains(blacklistedCountries, country)
}

func (f *IpFilter) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	ip := net.ParseIP(remoteIP)

	breakLoop := isIpInWhitelistedNetwork(ip) ||
		isIpWhitelisted(remoteIP) || google.IsGoogleBot(ip)
	if breakLoop {
		return BreakLoopResult
	}

	resolvedCountry := country.ResolveCountryByIP(remoteIP)

	if resolvedCountry != "" {
		if isCountryAllowed(resolvedCountry) {
			return BreakLoopResult
		}

		// Check if country is blacklisted
		if isCountryBlacklisted(resolvedCountry) {
			result := AbortRequestResult
			result.AbortHandler = methods.ForbiddenCountry(resolvedCountry, remoteIP)
			return result
		}

		return PassToNext
	}

	// cannot detect country
	// pass to next filter
	return PassToNext
}
