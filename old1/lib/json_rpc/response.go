package json_rpc

type Response struct {
	Id      interface{} `json:"id"`
	Result  interface{} `json:"result"`
	Success bool        `json:"success"`
	Reason  interface{} `json:"reason,omitempty"`
}

type ErrorResponse struct {
	Id    interface{} `json:"id"`
	Error Error       `json:"error"`
}
