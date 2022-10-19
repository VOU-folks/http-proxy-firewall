package json_rpc

func InvalidMethod(id interface{}) *ErrorResponse {
	return &ErrorResponse{
		Id: id,
		Error: Error{
			Type:    -32600,
			Message: "Invalid Method. Non-empty string required",
		},
	}
}
