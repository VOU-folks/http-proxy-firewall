package http

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"

	"http-proxy-firewall/lib/firewall/methods"
)

func shouldRecover(c *fiber.Ctx) {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic: %v | IP: %s | Host: %s | Path: %s\n",
			r, c.IP(), c.Hostname(), c.Path())
		methods.Refresh(c)
	}
}

func setForwardingHeaders(c *fiber.Ctx) string {
	proto := "http"
	if c.Protocol() == "https" {
		proto = "https"
	}
	host := c.Hostname()

	c.Request().Header.Set("X-Forwarded-Host", host)
	c.Request().Header.Set("X-Forwarded-Proto", proto)
	c.Request().Header.Set("Host", host)

	return proto
}

func setSecurityHeaders(c *fiber.Ctx, proto string) {
	c.Response().Header.Del("Server")
	if proto == "https" {
		c.Set("Strict-Transport-Security", "max-age=0")
		c.Set("Connection", "close")
	}
}

func ReverseProxy(targetServer string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer shouldRecover(c)

		proto := setForwardingHeaders(c)

		// Use Fiber's proxy middleware
		url := "http://" + targetServer
		if err := proxy.Do(c, url); err != nil {
			log.Printf("ErrorHandler in ReverseProxy: %s\n", err.Error())
			return err
		}

		setSecurityHeaders(c, proto)
		return nil
	}
}

// ProxyConfig creates a configured proxy with custom settings
func ProxyConfig(targetServer string) fiber.Handler {
	config := proxy.Config{
		Servers: []string{"http://" + targetServer},
		ModifyRequest: func(c *fiber.Ctx) error {
			setForwardingHeaders(c)
			return nil
		},
		ModifyResponse: func(c *fiber.Ctx) error {
			proto := "http"
			if c.Protocol() == "https" {
				proto = "https"
			}
			setSecurityHeaders(c, proto)
			return nil
		},
		Timeout: 10 * time.Minute,
	}

	return proxy.Balancer(config)
}
