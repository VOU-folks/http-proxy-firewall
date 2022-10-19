package structs

import (
	"net"
)

type Context struct {
	Id          string
	Connection  *net.TCPConn
	Connections *Connections
}
