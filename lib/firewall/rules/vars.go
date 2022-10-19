package rules

import (
	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/firewall/methods"
)

var BreakLoopResult = FilterResult{
	Error:     nil,
	Passed:    true,
	BreakLoop: true,
}

var PassToNext = FilterResult{
	Error:     nil,
	Passed:    true,
	BreakLoop: false,
}

var AbortRequestResult = FilterResult{
	Error:        nil,
	Passed:       false,
	BreakLoop:    false,
	AbortHandler: methods.Forbidden,
}
