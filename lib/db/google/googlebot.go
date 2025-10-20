package google

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var googlebotIPNetworkStorageDuration = time.Hour * 24
var redisTimeout = time.Second * 5
var httpClient = &http.Client{
	Timeout: time.Second * 30,
}
var googlebotIPStorage *GooglebotIPStorage
var googlebotIPStorageClient *GooglebotIPStorageClient

type GooglebotIPRecord = string

type GooglebotIPStorage struct {
	records  []GooglebotIPRecord
	networks []net.IPNet
	mx       sync.RWMutex
}

type GooglebotIPStorageClient struct {
	client    *redis.Client
	enabled   bool
	connected bool
	mx        sync.RWMutex
}

func EnableRedisClient(enable bool) {
	googlebotIPStorageClient.mx.Lock()
	googlebotIPStorageClient.enabled = enable
	googlebotIPStorageClient.mx.Unlock()
}

func (c *GooglebotIPStorageClient) StorageKey() string {
	return "GOOGLEBOT:NETWORK"
}

func (c *GooglebotIPStorageClient) IsActive() bool {
	c.mx.RLock()
	result := c.enabled && c.client != nil && c.connected
	c.mx.RUnlock()
	return result
}

func (c *GooglebotIPStorageClient) Start() {
	c.connected = false

	go func() {
		for {
			if c.enabled {
				ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
				_, err := c.client.Ping(ctx).Result()
				cancel()
				c.mx.Lock()
				c.connected = err == nil
				c.mx.Unlock()

				restoreFromStorageServer()
			}

			time.Sleep(time.Second * 10)
		}
	}()
}

func (c *GooglebotIPStorageClient) Store(records []GooglebotIPRecord) {
	if !c.IsActive() {
		return
	}

	data := strings.Join(records, ", ")
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	_, err := c.client.SetEX(ctx, c.StorageKey(), data, googlebotIPNetworkStorageDuration).Result()
	if err != nil {
		log.Println("GooglebotIPStorageClient.Store", err.Error())
	}
}

func (c *GooglebotIPStorageClient) Get() []GooglebotIPRecord {
	records := make([]GooglebotIPRecord, 0)

	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	data, _ := c.client.Get(ctx, c.StorageKey()).Result()
	if data != "" {
		records = strings.Split(data, ", ")
	}

	return records
}

func init() {
	googlebotIPStorage = &GooglebotIPStorage{
		records:  make([]GooglebotIPRecord, 0),
		networks: make([]net.IPNet, 0),
		mx:       sync.RWMutex{},
	}

	// Calculate optimal pool size: at least 10, or 4x CPU cores
	poolSize := runtime.NumCPU() * 4
	if poolSize < 10 {
		poolSize = 10
	}

	googlebotIPStorageClient = &GooglebotIPStorageClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:        "redis:6379",
				Password:    "",
				DB:          0,
				PoolSize:    poolSize,
				PoolTimeout: time.Second * 10,
			},
		),
		enabled: true,
		mx:      sync.RWMutex{},
	}

	googlebotIPStorageClient.Start()

	go getGoogleBotIPs()
}

type IPPrefix struct {
	IPv4Prefix string `json:"ipv4Prefix"`
	IPv6Prefix string `json:"ipv6Prefix"`
}

type IPRanges struct {
	Prefixes []IPPrefix `json:"prefixes"`
}

func restoreFromStorageServer() {
	if !googlebotIPStorageClient.IsActive() {
		return
	}

	// Check if records already exist (with lock)
	googlebotIPStorage.mx.RLock()
	hasRecords := len(googlebotIPStorage.records) > 0
	googlebotIPStorage.mx.RUnlock()

	if hasRecords {
		return
	}

	// Get records from Redis
	records := googlebotIPStorageClient.Get()
	if len(records) == 0 {
		return
	}

	// Parse networks
	networks := make([]net.IPNet, 0, len(records))
	var network *net.IPNet
	var err error
	for _, record := range records {
		_, network, err = net.ParseCIDR(record)

		if err != nil {
			log.Println("Cidr:", record, "parse error:", err.Error())
			continue
		}

		networks = append(networks, *network)
	}

	// Store atomically
	googlebotIPStorage.mx.Lock()
	googlebotIPStorage.records = records
	googlebotIPStorage.networks = networks
	googlebotIPStorage.mx.Unlock()
}

func getGoogleBotIPs() {
	for {
		var records = make([]GooglebotIPRecord, 0, 50)
		var networks = make([]net.IPNet, 0, 50)
		var googleIPRanges IPRanges

		resp, err := httpClient.Get("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
		if err != nil {
			log.Printf("Request Failed: %s", err)
			time.Sleep(time.Minute * 5) // Retry after 5 minutes on error
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Always close response body
		if err != nil {
			log.Printf("Failed to read IP ranges: %s", err)
			time.Sleep(time.Minute * 5) // Retry after 5 minutes on error
			continue
		}

		err = json.Unmarshal(body, &googleIPRanges)
		if err != nil {
			log.Printf("Failed to unmarshal IP ranges: %s", err)
			time.Sleep(time.Minute * 5) // Retry after 5 minutes on error
			continue
		}

		for _, prefix := range googleIPRanges.Prefixes {
			var network *net.IPNet

			if prefix.IPv4Prefix != "" {
				records = append(records, prefix.IPv4Prefix)
				_, network, err = net.ParseCIDR(prefix.IPv4Prefix)
				if err != nil {
					log.Println("Cidr:", prefix.IPv4Prefix, "parse error:", err.Error())
				} else if network != nil {
					networks = append(networks, *network)
				}
			}
			if prefix.IPv6Prefix != "" {
				records = append(records, prefix.IPv6Prefix)
				_, network, err = net.ParseCIDR(prefix.IPv6Prefix)
				if err != nil {
					log.Println("Cidr:", prefix.IPv6Prefix, "parse error:", err.Error())
				} else if network != nil {
					networks = append(networks, *network)
				}
			}
		}

		// putting ip range information to memory storage...
		googlebotIPStorage.mx.Lock()
		googlebotIPStorage.records = records
		googlebotIPStorage.networks = networks
		googlebotIPStorage.mx.Unlock()

		// ... and updating persistent storage
		googlebotIPStorageClient.Store(records)

		time.Sleep(googlebotIPNetworkStorageDuration)
	}
}

func IsGoogleBot(ip net.IP) bool {
	googlebotIPStorage.mx.RLock()
	defer googlebotIPStorage.mx.RUnlock()

	for _, network := range googlebotIPStorage.networks {
		if network.Contains(ip) {
			log.Println("IP: " + ip.String() + " is GoogleBot")
			return true
		}
	}

	return false
}
