package rules

import (
	"net/http"
	"strings"
)

var blacklistedIps = []string{
	"20.204.80.",
	"40.66.42.",
	"95.108.213.",
	"216.244.66.",
}

func DenyBlacklistedIps(req *http.Request, res http.ResponseWriter, remoteAddr string) bool {
	for _, ip := range blacklistedIps {
		if strings.Index(remoteAddr, ip) == 0 {
			res.WriteHeader(403)
			_, _ = res.Write([]byte("Forbidden"))
			return true
		}
	}

	return false
}
