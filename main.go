package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/firewall"
	"http-proxy-firewall/lib/firewall/methods"
	"http-proxy-firewall/lib/http"
)

func main() {
	var addr = flag.String("addr", "0.0.0.0:80", "Proxy address")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	app := gin.New()
	app.Use(gin.Recovery())

	app.GET("/_pf/_status", http.Status)

	app.Use(firewall.Handler)
	app.Use(http.ReverseProxy("127.0.0.1:8008"))

	// app.NoRoute(methods.NotFound)
	app.NoMethod(methods.NotFound)

	startListeners(addr, app)
}

func startListeners(addr *string, app *gin.Engine) {
	httpServer := http.CreateHttpServer(*addr)
	httpServer.Handler = app

	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("httpServer.ListenAndServe:", err.Error())
	}
	log.Println("Proxy-firewall listening at", *addr)
}
