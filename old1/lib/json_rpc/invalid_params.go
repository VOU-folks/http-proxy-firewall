package json_rpc

func InvalidParams(id interface{}) *ErrorResponse {
	return &ErrorResponse{
		Id: id,
		Error: Error{
			Type:    -32602,
			Message: "Invalid Params",
		},
	}
}
