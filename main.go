package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/fx"
	"golang.org/x/crypto/acme/autocert"

	"http-proxy-firewall/lib/firewall"
	"http-proxy-firewall/lib/firewall/methods"
	proxyhttp "http-proxy-firewall/lib/http"
	"http-proxy-firewall/lib/metrics"
	"http-proxy-firewall/lib/utils"
)

// Config holds application configuration
type Config struct {
	Listen         string
	ProxyTo        string
	MetricsEnabled bool
	SilentMode     bool
	EnableRedis    bool
}

// NewConfig creates configuration from command line flags
func NewConfig() *Config {
	listen := flag.String("listen", "0.0.0.0:80", "Address to listen at (default 0.0.0.0:80)")
	proxyTo := flag.String("proxy-to", "127.0.0.1:8008", "Address of remote server to proxy (default 127.0.0.1:8008)")
	metricsEnabled := flag.Bool("metrics", false, "Enable metrics (default false)")
	silentMode := flag.Bool("silent", true, "Disable verbosity, log only errors (default true)")
	enableRedis := flag.Bool("enable-redis", false, "Enable redis server usage for in memory objects (default false)")
	flag.Parse()

	config := &Config{
		Listen:         *listen,
		ProxyTo:        *proxyTo,
		MetricsEnabled: *metricsEnabled,
		SilentMode:     *silentMode,
		EnableRedis:    *enableRedis,
	}

	log.Println("listen =", config.Listen)
	log.Println("proxy-to =", config.ProxyTo)
	log.Println("metrics =", config.MetricsEnabled)
	log.Println("silent =", config.SilentMode)
	log.Println("enable-redis =", config.EnableRedis)

	firewall.EnableRedis(config.EnableRedis)

	return config
}

// AutocertManager provides the autocert manager
type AutocertManager struct {
	Manager *autocert.Manager
}

// NewAutocertManager creates a new autocert manager
func NewAutocertManager() *AutocertManager {
	cwd, _ := os.Getwd()
	cacheDir := cwd + "/.cache"
	email := utils.GetEnv("LETSENCRYPT_EMAIL")
	log.Println("LETSENCRYPT_EMAIL =", email)

	return &AutocertManager{
		Manager: &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(cacheDir),
			Email:  email,
		},
	}
}

// NewFiberApp creates and configures the Fiber application
func NewFiberApp(config *Config) *fiber.App {
	// Configure Fiber
	fiberConfig := fiber.Config{
		DisableStartupMessage: config.SilentMode,
		ServerHeader:          "",
		AppName:               "Proxy-Firewall",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			log.Printf("Error: %v", err)
			return c.Status(code).SendString(err.Error())
		},
		ReadTimeout:  10 * 60 * 1000, // 10 minutes in milliseconds
		WriteTimeout: 10 * 60 * 1000,
		IdleTimeout:  60 * 1000,
	}

	app := fiber.New(fiberConfig)

	// Recovery middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: !config.SilentMode,
	}))

	// Metrics middleware if enabled
	if config.MetricsEnabled {
		log.Println("Attaching metrics monitor")
		app.Use(metrics.MetricsMiddleware())
		app.Get("/__system__/__metrics__", metrics.MetricsHandler())
	}

	// Firewall middlewares
	app.Use(firewall.Handler)
	app.Use(firewall.BotHandler)

	// Reverse proxy
	app.Use(proxyhttp.ReverseProxy(config.ProxyTo))

	// 404 handler
	app.Use(func(c *fiber.Ctx) error {
		return methods.NotFound(c)
	})

	return app
}

// StartHTTPServer starts the HTTP server
func StartHTTPServer(
	lc fx.Lifecycle,
	app *fiber.App,
	config *Config,
	acm *AutocertManager,
) {
	httpAddr := config.Listen

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				// HTTP server with ACME handler for Let's Encrypt
				log.Printf("Starting HTTP server on %s", httpAddr)

				// Create a separate app for HTTP that handles ACME challenges
				// and forwards everything else to the main app
				httpApp := fiber.New(fiber.Config{
					DisableStartupMessage: config.SilentMode,
				})

				// Add ACME handler
				httpApp.Use(adaptor.HTTPHandler(acm.Manager.HTTPHandler(nil)))

				if err := httpApp.Listen(httpAddr); err != nil {
					log.Fatalf("HTTP server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping HTTP server...")
			return app.Shutdown()
		},
	})
}

// StartHTTPSServer starts the HTTPS server
func StartHTTPSServer(
	lc fx.Lifecycle,
	app *fiber.App,
	config *Config,
	acm *AutocertManager,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Println("Starting HTTPS server on :443")

				ln, err := tls.Listen("tcp", ":443", &tls.Config{
					GetCertificate: acm.Manager.GetCertificate,
					MinVersion:     tls.VersionTLS12,
					NextProtos:     []string{"http/1.1"},
				})
				if err != nil {
					log.Fatalf("HTTPS listener error: %v", err)
					return
				}

				if err := app.Listener(ln); err != nil {
					log.Fatalf("HTTPS server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping HTTPS server...")
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			NewConfig,
			NewAutocertManager,
			NewFiberApp,
		),
		fx.Invoke(
			StartHTTPServer,
			StartHTTPSServer,
		),
	)

	app.Run()
}
