package rules

import (
	"net/http"
	"strings"

	"http-proxy-firewall/helpers"
)

var allowedMethods = []string{
	"get", "post", "put", "delete", "patch", "options", "head",
}

func CheckIfMethodAllowed(req *http.Request, res http.ResponseWriter) bool {
	if helpers.Contains[string](allowedMethods, strings.ToLower(req.Method)) {
		return true
	}

	res.WriteHeader(405)
	_, _ = res.Write([]byte("Method Not Allowed"))
	return false
}
