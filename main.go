package main

import (
	"crypto/tls"
	"flag"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
	"http-proxy-firewall/lib/firewall"
	"http-proxy-firewall/lib/firewall/methods"
	"http-proxy-firewall/lib/http"
	"http-proxy-firewall/lib/metrics"
	"http-proxy-firewall/lib/utils"
	"log"
	"os"
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

	go startTlsListeners(listen, app)
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

	app.Use(firewall.Handler)
	app.Use(http.ReverseProxy(proxyTo))

	app.NoMethod(methods.NotFound)

	return app
}

var autocertManager autocert.Manager

func init() {
	cwd, _ := os.Getwd()
	cacheDir := cwd + "/.cache"
	autocertManager = autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(cacheDir),
		Email:  utils.GetEnv("LETSENCRYPT_EMAIL"),
	}
}

func startListeners(addr *string, app *gin.Engine) {
	httpServer := http.CreateHttpServer(*addr)
	httpServer.Handler = autocertManager.HTTPHandler(app)

	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("httpServer.ListenAndServe:", err.Error())
	}
	log.Println("Proxy-firewall listening at", *addr)
}

func startTlsListeners(addr *string, app *gin.Engine) {
	httpsServer := http.CreateHttpsServer(
		":https",
		&tls.Config{
			GetCertificate: autocertManager.GetCertificate,
			MinVersion:     tls.VersionTLS11, // for some unfortunately old clients
			NextProtos:     []string{"http/1.1"},
		},
	)
	httpsServer.Handler = app

	err := httpsServer.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal("httpsServer.ListenAndServe:", err.Error())
	}
	log.Println("Proxy-firewall listening at", *addr)
}
