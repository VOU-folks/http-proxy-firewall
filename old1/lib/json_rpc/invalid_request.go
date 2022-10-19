package json_rpc

func InvalidRequest(id interface{}) *ErrorResponse {
	return &ErrorResponse{
		Id: id,
		Error: Error{
			Type:    -32600,
			Message: "Invalid Request",
		},
	}
}
