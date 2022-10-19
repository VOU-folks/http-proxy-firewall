package handlers

import (
	"http-proxy-firewall/apps/tcp_server/lib/methods"
	"http-proxy-firewall/lib/log"
)

func HandleMethod(id string, method string, params interface{}) (bool, interface{}, error) {
	log.WithFields(log.Fields{"id": id, "method": method, "params": params}).Debug("Method")

	var result interface{}
	var handled = true
	var err error

	switch method {
	case "ping":
		result = methods.Ping(id, params)
		break

	case "time":
		result = methods.Time(id, params)
		break

	default:
		handled = false
	}

	return handled, result, err
}
