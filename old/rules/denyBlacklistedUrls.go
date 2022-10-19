package rules

import (
	"net/http"
	"strings"
)

var blacklistedUrls = []string{
	"/azercell_reg.php",
}

func DenyBlacklistedUrls(req *http.Request, res http.ResponseWriter) bool {
	url := req.URL.String()

	for _, item := range blacklistedUrls {
		if strings.Contains(url, item) {
			res.WriteHeader(403)
			_, _ = res.Write([]byte("Forbidden"))
			return true
		}
	}

	return false
}