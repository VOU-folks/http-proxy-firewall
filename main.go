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
	var err error
	var addr = flag.String("addr", "0.0.0.0:80", "Proxy address")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	app := gin.New()
	app.Use(gin.Recovery())
	app.Use(firewall.Handler)
	app.Use(http.ReverseProxy("127.0.0.1:8008"))

	// app.NoRoute(methods.NotFound)
	app.NoMethod(methods.NotFound)

	server := http.CreateHttpServer(*addr)
	server.Handler = app

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe:", err.Error())
	}
	log.Println("Proxy-firewall listening on", *addr)
}
