package http

import (
	"crypto/tls"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 10 * time.Minute
	writeTimeout      = 10 * time.Minute
	idleTimeout       = 60 * time.Second
)

func createServer(listenAt string, tlsConfig *tls.Config) *http.Server {
	return &http.Server{
		Addr:              listenAt,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		TLSConfig:         tlsConfig,
	}
}

func CreateHttpServer(listenAt string) *http.Server {
	return createServer(listenAt, nil)
}

func CreateHttpsServer(listenAt string, config *tls.Config) *http.Server {
	return createServer(listenAt, config)
}
