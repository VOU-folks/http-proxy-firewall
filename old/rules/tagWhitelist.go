package rules

import (
	"mime"
	"net/http"
	"strings"

	"http-proxy-firewall/helpers"
)

var whitelistIps = []string{
	"127.0.0.1",
	"10.0.0.1",
	"10.0.0.2",
	"10.0.0.3",
	"10.0.0.4",
	"49.12.20.252",
	"162.55.45.158",

	"185.22.155.63",
	"31.31.198.237",
	"167.233.11.197",

	// external
	"136.243.135.116", // bro1
	"31.31.198.237",
}

var whitelistIpPrefixes = []string{
	"5.",
	"92.250.102.",
}

func TagWhitelist(req *http.Request, remoteAddr string) bool {
	isWhitelisted := false

	req.Header.Set("Is-Whitelisted", "false")

	if helpers.Contains[string](whitelistIps, remoteAddr) {
		req.Header.Set("Is-Whitelisted", "true")
		isWhitelisted = true
		return isWhitelisted
	}

	for _, prefix := range whitelistIpPrefixes {
		if strings.Index(remoteAddr, prefix) == 0 {
			req.Header.Set("Is-Whitelisted", "true")
			isWhitelisted = true
			return isWhitelisted
		}
	}

	pos := strings.LastIndex(req.URL.Path, ".")
	if pos > 0 {
		ext := req.URL.Path[pos:len(req.URL.Path)]
		if ext == ".css" || ext == ".js" {
			req.Header.Set("Is-Whitelisted", "true")
			isWhitelisted = true
			return isWhitelisted
		}

		mimeType := mime.TypeByExtension(ext)
		if strings.Contains(mimeType, "image/") || strings.Contains(mimeType, "video/") {
			req.Header.Set("Is-Whitelisted", "true")
			isWhitelisted = true
			return isWhitelisted
		}
	}

	return isWhitelisted
}
