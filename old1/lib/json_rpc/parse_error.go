package json_rpc

var parseError = &ErrorResponse{
	Id: nil,
	Error: Error{
		Type:    -32700,
		Message: "Parse error",
	},
}

func ParseError() *ErrorResponse {
	return parseError
}
