package json_rpc

var invalidId = &ErrorResponse{
	Id: nil,
	Error: Error{
		Type:    -32600,
		Message: "Invalid Id. Non-empty string required",
	},
}

func InvalidId() *ErrorResponse {
	return invalidId
}
