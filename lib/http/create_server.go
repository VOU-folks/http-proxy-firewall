package http

import (
	"crypto/tls"
	"net/http"
	"time"
)

func CreateHttpServer(listenAt string) *http.Server {
	return &http.Server{
		Addr:              listenAt,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
}

func CreateHttpsServer(listenAt string, config *tls.Config) *http.Server {
	return &http.Server{
		Addr:              listenAt,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       60 * time.Second,
		TLSConfig:         config,
	}
}
