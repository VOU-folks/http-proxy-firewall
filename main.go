package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/firewall"
	"http-proxy-firewall/lib/firewall/methods"
	"http-proxy-firewall/lib/http"
	"http-proxy-firewall/lib/metrics"
)

func main() {
	var listen = flag.String("listen", "0.0.0.0:80", "Address to listen at (default 0.0.0.0:80)")
	var proxyTo = flag.String("proxy-to", "127.0.0.1:8008", "Address of remote server to proxy (default 127.0.0.1:8008)")
	var metricsEnabled = flag.Bool("metrics", false, "Enable metrics (default false)")
	var silentMode = flag.Bool("silent", true, "Disable verbosity, log only errors (default true)")
	var enableRedis = flag.Bool("enable-redis", false, "Enable redis server usage for in memory objects (default false) like cookies, ip-country")
	flag.Parse()

	log.Println("listen =", *listen)
	log.Println("proxy-to =", *proxyTo)
	log.Println("metrics =", *metricsEnabled)
	log.Println("silent =", *silentMode)
	log.Println("enable-redis =", *enableRedis)

	firewall.EnableRedis(*enableRedis)

	app := createAppInstance(*proxyTo, *metricsEnabled, *silentMode)

	startListeners(listen, app)
}

func createAppInstance(proxyTo string, withMetrics bool, silentMode bool) *gin.Engine {

	if silentMode == true {
		gin.SetMode(gin.ReleaseMode)
	}

	app := gin.New()
	app.Use(gin.Recovery())

	if withMetrics == true {
		log.Println("Attaching metrics monitor")
		monitor := metrics.GetMonitor()
		monitor.Use(app)
	}

	app.Use(func(c *gin.Context) {
		c.Header("Connection", "close")
		c.Next()
	})

	app.Use(firewall.Handler)
	app.Use(http.ReverseProxy(proxyTo))

	// app.NoRoute(methods.NotFound)
	app.NoMethod(methods.NotFound)

	return app
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
