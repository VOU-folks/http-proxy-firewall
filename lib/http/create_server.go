package http

import (
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
