package http

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/firewall/methods"
)

type TransportStorage struct {
	transports []*http.Transport
	seq        int
	size       int
	mx         sync.Mutex
}

func (ts *TransportStorage) Init() {
	transportsCount := runtime.NumCPU() * 2

	ts.transports = make([]*http.Transport, transportsCount)
	ts.mx = sync.Mutex{}
	ts.size = transportsCount
	for n := 0; n < ts.size; n++ {
		ts.transports[n] = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 10 * time.Minute,
			IdleConnTimeout:       1 * time.Minute,
			DisableKeepAlives:     false,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       0,
			ForceAttemptHTTP2:     false,
		}
	}

	log.Println("Created ", ts.size, " http transports")
}

func (ts *TransportStorage) Get() *http.Transport {
	var transport *http.Transport

	ts.mx.Lock()

	if ts.seq == ts.size {
		ts.seq = 0
	}
	transport = ts.transports[ts.seq]
	ts.seq++

	ts.mx.Unlock()

	return transport
}

var transportStorage *TransportStorage

func init() {
	transportStorage = &TransportStorage{}
	transportStorage.Init()
}

func requestDirector(req *http.Request, targetServer string, host string) func(req *http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = targetServer

		req.Header.Set("Host", host)
	}
}

func errorHandler(writer http.ResponseWriter, request *http.Request, err error) {
	if !strings.Contains(err.Error(), "context canceled") {
		log.Println("ErrorHandler in ReverseProxy", err.Error())
	}
}

func shouldRecover(c *gin.Context) {
	if r := recover(); r != nil {
		fmt.Println("Recovered from", r)
		methods.NotFound(c)
	}
}

func ReverseProxy(targetServer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer shouldRecover(c)

		host := c.Request.Host

		proxy := &httputil.ReverseProxy{
			Director:     requestDirector(c.Request, targetServer, host),
			Transport:    transportStorage.Get(),
			ErrorHandler: errorHandler,
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
