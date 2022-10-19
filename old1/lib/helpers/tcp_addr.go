package helpers

import (
	"net"
	"strconv"
)

func GetTCPAddr() *net.TCPAddr {
	HOST := GetEnv("TCP_SERVER_HOST")
	if HOST == "" {
		HOST = GetEnvOr("HOST", DEFAULT_HOST)
	}

	PORT := GetEnv("TCP_SERVER_PORT")
	if PORT == "" {
		PORT = GetEnvOr("PORT", DEFAULT_PORT)
	}

	Port, _ := strconv.Atoi(PORT)

	var tcpAddr net.TCPAddr
	tcpAddr.IP = net.ParseIP(HOST)
	tcpAddr.Port = Port

	return &tcpAddr
}
