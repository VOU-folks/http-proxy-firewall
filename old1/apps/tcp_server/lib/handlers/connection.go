package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net"

	"http-proxy-firewall/apps/tcp_server/lib/structs"
	"http-proxy-firewall/lib/json_rpc"
	"http-proxy-firewall/lib/log"
)

func ParseClientPayload(ctx *structs.Context, payload []byte, request *json_rpc.Request) error {
	err := json.Unmarshal(payload, request)
	if err != nil {
		log.Error(err.Error())
		msg, _ := json.Marshal(json_rpc.ParseError())
		_, _ = ctx.Connection.Write(append(msg, "\n"...))
		return err
	}

	if !request.ValidId() {
		rpcError := json_rpc.InvalidId()
		msg, _ := json.Marshal(rpcError)
		_, _ = ctx.Connection.Write(append(msg, "\n"...))
		return errors.New(rpcError.Error.Message)
	}

	if !request.ValidMethod() {
		rpcError := json_rpc.InvalidMethod(request.Id)
		msg, _ := json.Marshal(rpcError)
		_, _ = ctx.Connection.Write(append(msg, "\n"...))
		return errors.New(rpcError.Error.Message)
	}

	return nil
}

func HandleConnection(ctx *structs.Context) {
	log.WithFields(
		log.Fields{
			"id":      ctx.Id,
			"address": ctx.Connection.RemoteAddr().String(),
		},
	).Info("Connection")

	server := CreateConnectionToServer()
	go io.Copy(server, ctx.Connection)
	io.Copy(ctx.Connection, server)

	ctx.Connections.Close(ctx.Id)
	ctx.Connections.Del(ctx.Id)
}

func CreateConnectionToServer() *net.TCPConn {
	var tcpAddr net.TCPAddr
	tcpAddr.IP = net.ParseIP("162.55.45.158")
	tcpAddr.Port = 80

	conn, err := net.DialTCP("tcp", nil, &tcpAddr)
	if err != nil {
		log.Error("Cannot connect to server.", err.Error())
		return nil
	}

	return conn
}
