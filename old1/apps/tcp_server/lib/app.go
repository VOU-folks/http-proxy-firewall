package lib

import (
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"http-proxy-firewall/apps/tcp_server/lib/handlers"
	"http-proxy-firewall/apps/tcp_server/lib/structs"
	"http-proxy-firewall/lib/constants"
	"http-proxy-firewall/lib/log"
)

type App struct {
	listener    *net.TCPListener
	connections *structs.Connections
}

func CreateApp() *App {
	app := &App{}
	app.connections = &structs.Connections{}
	app.connections.Init()

	return app
}

func (app *App) createListener(tcpAddr *net.TCPAddr) net.Listener {
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Error("Error listening:", err.Error())
		os.Exit(constants.ERR_TCP_LISTENER_START)
	}

	log.Println("Listening at tcp://" + tcpAddr.AddrPort().String())

	app.listener = listener

	return listener
}

func (app *App) acceptConnections() {
	for {
		if app.connections.Draining() || app.connections.Closed() {
			break
		}

		conn, err := app.listener.AcceptTCP()
		if err != nil {
			if app.connections.Draining() {
				break
			}
			log.Error("Error accepting:", err.Error())
			os.Exit(constants.ERR_TCP_LISTENER_ACCEPT)
		}

		_ = conn.SetKeepAlive(false)
		// _ = conn.SetKeepAlivePeriod(helpers.GetEnvAsDuration("TCP_KEEPALIVE_DURATION", "3s"))

		id, _ := app.connections.Add(conn)
		ctx := app.CreateContext(id)

		go handlers.HandleConnection(ctx)
	}
}

func (app *App) CreateContext(id string) *structs.Context {
	return &structs.Context{
		Id:          id,
		Connections: app.connections,
		Connection:  app.connections.Get(id),
	}
}

func (app *App) Listen(tcpAddr *net.TCPAddr) {
	app.createListener(tcpAddr)
	go app.acceptConnections()
}

func (app *App) WaitForCloseSignal(gracefulShutdownOnExit bool) {
	log.Println("Press Control-C to stop")

	c := make(chan os.Signal, 2)
	signal.Notify(
		c,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)

	sig := <-c
	log.WithFields(log.Fields{
		"sig": sig,
	}).Println("Got", sig, "signal")

	app.Shutdown(gracefulShutdownOnExit)

	runtime.Goexit()
}

func (app *App) Shutdown(graceful bool) {
	if graceful {
		log.Println("Draining connections")
		app.connections.Drain()

		log.Println("Closing listener")
		_ = app.listener.Close()

		time.Sleep(time.Microsecond)
	}

	log.Println("Bye (:")

	os.Exit(0)
}
