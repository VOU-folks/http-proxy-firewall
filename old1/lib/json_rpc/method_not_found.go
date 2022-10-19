package json_rpc

func MethodNotFound(id interface{}) *ErrorResponse {
	return &ErrorResponse{
		Id: id,
		Error: Error{
			Type:    -32601,
			Message: "Method not found",
		},
	}
}
