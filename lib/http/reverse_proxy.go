package http

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"

	"http-proxy-firewall/lib/firewall/methods"
)

func shouldRecover(c *fiber.Ctx) {
	if r := recover(); r != nil {
		fmt.Println(
			"Recovered from", r,
			c.IP(),
			c.Hostname(),
			c.Path(),
		)
		methods.Refresh(c)
	}
}

func ReverseProxy(targetServer string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer shouldRecover(c)

		proto := "http"
		if c.Protocol() == "https" {
			proto = "https"
			c.Set("Strict-Transport-Security", "max-age=0")
			c.Set("Connection", "close")
		}
		host := c.Hostname()

		// Set forwarding headers
		c.Request().Header.Set("X-Forwarded-Host", host)
		c.Request().Header.Set("X-Forwarded-Proto", proto)
		c.Request().Header.Set("Host", host)

		// Use Fiber's proxy middleware
		url := "http://" + targetServer
		if err := proxy.Do(c, url); err != nil {
			log.Println("ErrorHandler in ReverseProxy", err.Error())
			return err
		}

		// Modify response headers
		c.Response().Header.Del("Server")

		return nil
	}
}

// ProxyConfig creates a configured proxy with custom settings
func ProxyConfig(targetServer string) fiber.Handler {
	config := proxy.Config{
		Servers: []string{"http://" + targetServer},
		ModifyRequest: func(c *fiber.Ctx) error {
			proto := "http"
			if c.Protocol() == "https" {
				proto = "https"
			}
			host := c.Hostname()

			c.Request().Header.Set("X-Forwarded-Host", host)
			c.Request().Header.Set("X-Forwarded-Proto", proto)
			c.Request().Header.Set("Host", host)

			return nil
		},
		ModifyResponse: func(c *fiber.Ctx) error {
			c.Response().Header.Del("Server")
			if c.Protocol() == "https" {
				c.Set("Strict-Transport-Security", "max-age=0")
				c.Set("Connection", "close")
			}
			return nil
		},
		Timeout: 10 * time.Minute,
	}

	return proxy.Balancer(config)
}
