package json_rpc

func HandleError(id interface{}, err error) *ErrorResponse {
	return &ErrorResponse{
		Id: id,
		Error: Error{
			Type:    -32000,
			Message: err.Error(),
		},
	}
}
