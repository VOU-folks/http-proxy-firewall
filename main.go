package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"http-proxy-firewall/rules"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
	"Server",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

type proxy struct {
}

func earlyReject(req *http.Request, res http.ResponseWriter) bool {
	if !rules.CheckIfMethodAllowed(req, res) {
		return true
	}

	remoteAddr := strings.Trim(strings.Split(req.Header.Get("X-Forwarded-For"), ",")[0], " ")
	if remoteAddr == "" {
		remoteAddr = req.RemoteAddr
	}

	isWhitelisted := rules.TagWhitelist(req, remoteAddr)
	if isWhitelisted {
		return false
	}

	if rules.DenyBlacklistedIps(req, res, remoteAddr) {
		return true
	}

	/*
		if rules.DenyBadUserAgents(req, res) {
			return true
		}
	*/

	if rules.DenyBlacklistedUrls(req, res) {
		return true
	}

	if !rules.AllowByCountry(req, res, remoteAddr) {

	}

	if !rules.AuthorizeByCookie(req, res, remoteAddr) {
		return true
	}

	/*
		if rules.DoRateLimiting(req, res, remoteAddr) {
			rules.RejectSidCookie(req)
			return true
		}
	*/

	/*
		if rules.DoBlockByUrl(req, res) {
			return true
		}
	*/

	return false
}

type Counters struct {
	Rejected int
	Served   int
	mx       sync.Mutex
}

var counters = Counters{
	Rejected: 0,
	Served:   0,
	mx:       sync.Mutex{},
}

func (c *Counters) Served1() {
	c.mx.Lock()
	c.Served++
	c.mx.Unlock()
}

func (c *Counters) Rejected1() {
	c.mx.Lock()
	c.Rejected++
	c.mx.Unlock()
}

func (c *Counters) Print() string {
	var msg string

	c.mx.Lock()
	msg = fmt.Sprintf("Served: %d, Rejected: %d", c.Served, c.Rejected)
	c.mx.Unlock()

	log.Println(msg)

	return msg
}

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     20000,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	// http: Request.RequestURI can't be set in client requests.
	// http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	if req.URL.RequestURI() == "/favicon.ico" {
		wr.WriteHeader(204)
		return
	}

	if req.URL.RequestURI() == "/_server/_stats" {
		wr.Header().Set("Content-Type", "text/plain")
		wr.WriteHeader(200)
		wr.Write([]byte(counters.Print()))
		return
	}

	remoteAddr := strings.Trim(strings.Split(req.Header.Get("X-Forwarded-For"), ",")[0], " ")
	if remoteAddr == "" {
		remoteAddr = req.RemoteAddr
	}

	rejected := earlyReject(req, wr)
	if rejected == true {
		counters.Rejected1()
		return
	}
	defer counters.Served1()

	host, _, _ := net.SplitHostPort(req.Host)

	req.Header.Set("Host", host)
	delHopHeaders(req.Header)

	req.URL.Scheme = "http"
	req.URL.Host = "127.0.0.1:8008"

	resp, err := client.Do(req)
	if err != nil {
		if !strings.Contains(err.Error(), "context") {
			log.Println("ServeHTTP: ", err)
		}
		wr.WriteHeader(200)
		_, _ = wr.Write([]byte("OK"))
		return
	}
	defer resp.Body.Close()

	delHopHeaders(resp.Header)

	copyHeader(wr.Header(), resp.Header)
	wr.Header().Set("Server", "pf")
	wr.Header().Set("Connection", "close")
	wr.WriteHeader(resp.StatusCode)

	io.Copy(wr, resp.Body)
}

func main() {
	var addr = flag.String("addr", "0.0.0.0:80", "Proxy address")
	flag.Parse()

	handler := &proxy{}

	log.Println("Proxy-firewall listening on", *addr)
	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: time.Second * 1,
		// ReadTimeout:       time.Minute * 10,
		IdleTimeout:  time.Second * 3,
		WriteTimeout: time.Minute * 10,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
