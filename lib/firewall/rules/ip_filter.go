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
var ipWhitelist []string
var allowedCountries []string
var blacklistedCountries []string

func init() {
	// Initialize localhost whitelist
	_, network, err := net.ParseCIDR("127.0.0.1/8")
	if err != nil {
		log.Println("Failed to parse localhost CIDR:", err)
	} else {
		whitelistNetworks = append(whitelistNetworks, network)
	}

	// Load whitelisted networks from environment
	envWhitelistNets := strings.TrimSpace(utils.GetEnv("IP_FILTER_WHITELIST_NETWORKS"))
	if envWhitelistNets != "" {
		whitelistNets := strings.Split(envWhitelistNets, ",")
		for _, elem := range whitelistNets {
			trimmed := strings.TrimSpace(elem)
			if trimmed == "" {
				continue
			}
			_, network, err := net.ParseCIDR(trimmed)
			if err != nil {
				log.Println("Failed to parse network CIDR:", trimmed, err)
				continue
			}
			whitelistNetworks = append(whitelistNetworks, network)
		}
	}

	// Load whitelisted IPs from environment
	envWhitelist := strings.TrimSpace(utils.GetEnv("IP_FILTER_WHITELIST"))
	if envWhitelist != "" {
		whitelist := strings.Split(envWhitelist, ",")
		for _, elem := range whitelist {
			trimmed := strings.TrimSpace(elem)
			if trimmed != "" {
				ipWhitelist = append(ipWhitelist, trimmed)
			}
		}
	}

	// Load allowed countries from environment
	envCountries := strings.TrimSpace(utils.GetEnv("IP_FILTER_ALLOWED_COUNTRIES"))
	if envCountries != "" {
		countries := strings.Split(envCountries, ",")
		for _, elem := range countries {
			trimmed := strings.TrimSpace(elem)
			if trimmed != "" {
				allowedCountries = append(allowedCountries, trimmed)
			}
		}
	}

	// Load blacklisted countries from environment
	envBlacklistedCountries := strings.TrimSpace(utils.GetEnv("IP_FILTER_BLACKLISTED_COUNTRIES"))
	if envBlacklistedCountries != "" {
		blacklistedCountriesSlice := strings.Split(envBlacklistedCountries, ",")
		for _, elem := range blacklistedCountriesSlice {
			trimmed := strings.TrimSpace(elem)
			if trimmed != "" {
				blacklistedCountries = append(blacklistedCountries, trimmed)
			}
		}
	}

	log.Println("ip whitelist =", envWhitelist)
	log.Println("country whitelist =", envCountries)
	log.Println("country blacklist =", envBlacklistedCountries)
}

type IpFilter struct {
}

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

func isCountryAllowed(country string) bool {
	return slices.Contains(allowedCountries, country)
}

func isCountryBlacklisted(country string) bool {
	return slices.Contains(blacklistedCountries, country)
}

func (f *IpFilter) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	ip := net.ParseIP(remoteIP)

	// Check whitelists first (fastest path)
	if isIpInWhitelistedNetwork(ip) || isIpWhitelisted(remoteIP) || google.IsGoogleBot(ip) {
		return BreakLoopResult
	}

	// Resolve country if we have filtering rules
	if len(allowedCountries) > 0 || len(blacklistedCountries) > 0 {
		resolvedCountry := country.ResolveCountryByIP(remoteIP)

		if resolvedCountry != "" {
			// Whitelist has priority
			if len(allowedCountries) > 0 && isCountryAllowed(resolvedCountry) {
				return BreakLoopResult
			}

			// Check blacklist
			if len(blacklistedCountries) > 0 && isCountryBlacklisted(resolvedCountry) {
				return FilterResult{
					Error:        nil,
					Passed:       false,
					BreakLoop:    false,
					AbortHandler: methods.ForbiddenCountry(resolvedCountry, remoteIP),
				}
			}
		}
	}

	return PassToNext
}
