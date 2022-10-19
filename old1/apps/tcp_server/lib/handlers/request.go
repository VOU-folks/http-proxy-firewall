package handlers

import (
	"encoding/json"
	"sync/atomic"

	"http-proxy-firewall/apps/tcp_server/lib/structs"
	"http-proxy-firewall/lib/json_rpc"
	"http-proxy-firewall/lib/log"
)

func BeforeHandleRequest(ctx *structs.Context, req *json_rpc.Request) {
	atomic.AddUint64(&ctx.Connections.ActiveRequests, uint64(1))
	atomic.AddUint64(&ctx.Connections.TotalRequests, uint64(1))
	log.WithFields(log.Fields{"active": ctx.Connections.ActiveRequests, "total": ctx.Connections.TotalRequests}).Debug("BeforeHandleRequest")
}

func AfterHandleRequest(ctx *structs.Context, req *json_rpc.Request) {
	atomic.AddUint64(&ctx.Connections.ActiveRequests, ^uint64(0))
	log.WithFields(log.Fields{"active": ctx.Connections.ActiveRequests, "total": ctx.Connections.TotalRequests}).Debug("AfterHandleRequest")
}

func HandleRequest(ctx *structs.Context, req *json_rpc.Request) {
	log.WithFields(log.Fields{"id": req.Id, "method": req.Method, "params": req.Params}).Debug("Request")

	var result interface{}

	response := &json_rpc.Response{
		Id:      req.Id,
		Result:  nil,
		Success: true,
		Reason:  nil,
	}

	methodHandled, result, err := HandleMethod(req.Id.(string), req.Method, req.Params)

	if methodHandled {
		response.Result = result
		if err != nil {
			response.Success = false
			response.Reason = err.Error()
		}

		msg, _ := json.Marshal(response)
		_, _ = ctx.Connection.Write(append(msg, "\n"...))
		return
	}

	errorResponse := json_rpc.MethodNotFound(req.Id)
	msg, _ := json.Marshal(errorResponse)
	_, _ = ctx.Connection.Write(append(msg, "\n"...))
}
