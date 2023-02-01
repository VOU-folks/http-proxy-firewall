package google

import (
	"context"
	"encoding/json"
	"io/ioutil"
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
var googlebotIPStorage *GooglebotIPStorage
var googlebotIPStorageClient *GooglebotIPStorageClient

type GooglebotIPRecord = string

type GooglebotIPStorage struct {
	records  []GooglebotIPRecord
	networks []net.IPNet
	mx       sync.Mutex
}

type GooglebotIPStorageClient struct {
	client    *redis.Client
	enabled   bool
	connected bool
	mx        sync.Mutex
}

func EnableRedisClient(enable bool) {
	googlebotIPStorageClient.enabled = enable
}

func (c *GooglebotIPStorageClient) StorageKey() string {
	return "GOOGLEBOT:NETWORK"
}

func (c *GooglebotIPStorageClient) IsActive() bool {
	return c.enabled && c.client != nil && c.connected
}

func (c *GooglebotIPStorageClient) Start() {
	c.connected = false

	go func() {
		for {
			if c.enabled {
				_, err := c.client.Ping(context.Background()).Result()
				c.mx.Lock()
				c.connected = err == nil
				c.mx.Unlock()

				restoreFromStorageServer()
			}

			time.Sleep(time.Minute)
		}
	}()
}

func (c *GooglebotIPStorageClient) Store(records []GooglebotIPRecord) {
	if !c.IsActive() {
		return
	}

	data := strings.Join(records, ", ")
	_, err := c.client.SetEX(context.Background(), c.StorageKey(), data, googlebotIPNetworkStorageDuration).Result()
	if err != nil {
		log.Println("GooglebotIPStorageClient.Store", err.Error())
	}
}

func (c *GooglebotIPStorageClient) Get() []GooglebotIPRecord {
	records := make([]GooglebotIPRecord, 0)

	data, _ := c.client.Get(context.Background(), c.StorageKey()).Result()
	if data != "" {
		records = strings.Split(data, ", ")
	}

	return records
}

func init() {
	googlebotIPStorage = &GooglebotIPStorage{
		records:  make([]GooglebotIPRecord, 0),
		networks: make([]net.IPNet, 0),
		mx:       sync.Mutex{},
	}

	googlebotIPStorageClient = &GooglebotIPStorageClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:        "redis:6379",
				Password:    "",
				DB:          0,
				PoolSize:    runtime.NumCPU() - runtime.NumCPU()%3,
				PoolTimeout: time.Second * 10,
			},
		),
		enabled: false,
		mx:      sync.Mutex{},
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

	if len(googlebotIPStorage.records) > 0 {
		return
	}

	var err error

	googlebotIPStorage.records = googlebotIPStorageClient.Get()

	networks := make([]net.IPNet, 0)
	var network *net.IPNet
	for _, record := range googlebotIPStorage.records {
		_, network, err = net.ParseCIDR(record)

		if err != nil {
			log.Println("Cidr:", network, "parse error:", err.Error())
			continue
		}

		networks = append(networks, *network)
	}

	googlebotIPStorage.networks = networks
}

func getGoogleBotIPs() {
	var records = make([]GooglebotIPRecord, 0)
	var networks = make([]net.IPNet, 0)
	var googleIPRanges IPRanges

	for {
		resp, err := http.Get("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
		if err != nil {
			log.Printf("Request Failed: %s", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read IP ranges: %s", err)
			continue
		}

		_ = json.Unmarshal(body, &googleIPRanges)

		for _, prefix := range googleIPRanges.Prefixes {
			var network *net.IPNet

			if prefix.IPv4Prefix != "" {
				records = append(records, prefix.IPv4Prefix)
				_, network, err = net.ParseCIDR(prefix.IPv4Prefix)
				if err != nil {
					log.Println("Cidr:", prefix.IPv4Prefix, "parse error:", err.Error())
				}
			}
			if prefix.IPv6Prefix != "" {
				records = append(records, prefix.IPv6Prefix)
				_, network, err = net.ParseCIDR(prefix.IPv6Prefix)
				if err != nil {
					log.Println("Cidr:", prefix.IPv6Prefix, "parse error:", err.Error())
				}
			}

			if network != nil {
				networks = append(networks, *network)
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
	result := false

	googlebotIPStorage.mx.Lock()
	for _, network := range googlebotIPStorage.networks {
		if network.Contains(ip) {
			log.Println("IP: " + ip.String() + " is GoogleBot")
			result = true
			break
		}
	}
	googlebotIPStorage.mx.Unlock()

	return result
}
