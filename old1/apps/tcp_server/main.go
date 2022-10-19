package main

import (
	App "http-proxy-firewall/apps/tcp_server/lib"
	"http-proxy-firewall/lib/helpers"
	"http-proxy-firewall/lib/log"
)

func init() {
	helpers.LoadEnv()
	log.Init()
}

func main() {
	app := App.CreateApp()

	app.Listen(helpers.GetTCPAddr())

	app.WaitForCloseSignal(true)
}
