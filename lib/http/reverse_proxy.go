package http

import (
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var transport *http.Transport

func init() {
	transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 10 * time.Minute,
		IdleConnTimeout:       1 * time.Minute,
		DisableKeepAlives:     false,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       0,
		ForceAttemptHTTP2:     false,
	}
}

func requestDirector(req *http.Request, targetServer string, host string) func(req *http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = targetServer

		req.Header.Set("Host", host)
	}
}

func ReverseProxy(targetServer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host

		proxy := &httputil.ReverseProxy{
			Director:  requestDirector(c.Request, targetServer, host),
			Transport: transport,
			ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
				if !strings.Contains(err.Error(), "context canceled") {
					log.Println("ErrorHandler in ReverseProxy", err.Error())
				}

				// writer.WriteHeader(200)
				// writer.Header().Set("Content-Type", "text/html")
				// _, _ = writer.Write([]byte("<meta http-equiv=\"refresh\" content=\"1\">"))
			},
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
