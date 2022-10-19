package json_rpc

type Request struct {
	Id     interface{} `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

func (r *Request) ValidId() bool {
	switch r.Id.(type) {
	case string:
		return r.Id != ""
	}
	return false
}

func (r *Request) ValidMethod() bool {
	return r.Method != ""
}
