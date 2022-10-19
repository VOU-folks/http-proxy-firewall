package structs

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid"

	"http-proxy-firewall/lib/json_rpc"
	"http-proxy-firewall/lib/log"
)

type Client struct {
	socket *net.TCPConn
	open   bool
	mu     sync.Mutex
}

func (c *Client) Close() {
	c.mu.Lock()
	c.open = false
	if c.socket != nil {
		_ = c.socket.Close()
	}
	c.socket = nil
	c.mu.Unlock()
}

func CreateClient(socket *net.TCPConn) *Client {
	return &Client{
		open:   true,
		socket: socket,
		mu:     sync.Mutex{},
	}
}

type Connections struct {
	clients        map[string]*Client
	draining       bool
	closed         bool
	TotalRequests  uint64
	ActiveRequests uint64
	clientsMu      sync.Mutex
	drainingMu     sync.Mutex
	closingMu      sync.Mutex
}

func (cs *Connections) Init() *Connections {
	cs.draining = false
	cs.closed = false
	cs.clients = make(map[string]*Client)
	cs.TotalRequests = uint64(0)
	cs.ActiveRequests = uint64(0)
	cs.clientsMu = sync.Mutex{}
	cs.drainingMu = sync.Mutex{}
	cs.closingMu = sync.Mutex{}
	// cs.gcLoop()

	return cs
}

func (cs *Connections) eachClient(fn func(id string, client *Client)) {
	for id, client := range cs.clients {
		fn(id, client)
	}
}

func (cs *Connections) gcLoop() {
	go func() {
	GC:
		for {
			cs.eachClient(func(id string, client *Client) {
				if cs.Draining() {
					return
				}

				if !client.open {
					cs.Del(id)
				}
			})

			if cs.Draining() {
				break GC
			}

			time.Sleep(time.Minute)
		}
	}()
}

func (cs *Connections) Add(socket *net.TCPConn) (string, *net.TCPConn) {
	id := gonanoid.MustID(32)

	cs.clientsMu.Lock()
	cs.clients[id] = CreateClient(socket)
	cs.clientsMu.Unlock()

	return id, socket
}

func (cs *Connections) Get(id string) *net.TCPConn {
	return cs.clients[id].socket
}

func (cs *Connections) client(id string) *Client {
	cs.clientsMu.Lock()
	client := cs.clients[id]
	cs.clientsMu.Unlock()

	return client
}

func (cs *Connections) Del(id string) *Connections {
	cs.clientsMu.Lock()
	delete(cs.clients, id)
	cs.clientsMu.Unlock()

	log.Info("Connection deleted")
	return cs
}

func (cs *Connections) Broadcast(data []byte) *Connections {
	cs.eachClient(func(id string, client *Client) {
		client.mu.Lock()
		if client.socket != nil {
			_, _ = client.socket.Write(data)
		}
		client.mu.Unlock()
	})

	return cs
}

func (cs *Connections) ActiveConnections() int {
	var c = 0

	cs.eachClient(func(id string, client *Client) {
		if client.open {
			c++
		}
	})

	return c
}

func (cs *Connections) Drain() *Connections {
	cs.drainingMu.Lock()
	cs.draining = true
	cs.drainingMu.Unlock()

	msg, _ := json.Marshal(json_rpc.Drain)

	for {
		cs.Broadcast(
			append(msg, "\n"...),
		)

		if cs.ActiveRequests == 0 {
			cs.CloseAll()
			break
		}

		if cs.ActiveConnections() == 0 {
			cs.closed = true
			break
		}

		time.Sleep(time.Second)
	}

	return cs
}

func (cs *Connections) Close(id string) {
	client := cs.client(id)
	if client != nil {
		log.WithFields(
			log.Fields{
				"id":      id,
				"address": client.socket.RemoteAddr().String(),
			},
		).Info("Closing connection")
		client.Close()
	}
}

func (cs *Connections) CloseAll() {
	cs.eachClient(func(id string, client *Client) {
		client.Close()
	})

	cs.closingMu.Lock()
	cs.closed = true
	cs.closingMu.Unlock()
}

func (cs *Connections) Draining() bool {
	cs.drainingMu.Lock()
	result := cs.draining == true
	cs.drainingMu.Unlock()

	return result
}

func (cs *Connections) Closed() bool {
	cs.closingMu.Lock()
	result := cs.closed == true
	cs.closingMu.Unlock()

	return result
}
