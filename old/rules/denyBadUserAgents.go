package rules

import (
	"net/http"
)

var badUserAgentParts = []string{
	"curl",
}

func DenyBadUserAgents(req *http.Request, res http.ResponseWriter) bool {
	userAgent := req.Header.Get("User-Agent")

	if userAgent == "" {
		res.WriteHeader(403)
		_, _ = res.Write([]byte("Forbidden"))
		return true
	}

	// for _, part := range badUserAgentParts {
	// 	if strings.Contains(userAgent, part) {
	// 		res.WriteHeader(403)
	// 		_, _ = res.Write([]byte("Forbidden"))
	// 		return true
	// 	}
	// }

	return false
}
