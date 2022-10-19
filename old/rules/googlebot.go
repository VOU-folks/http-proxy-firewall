package rules

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var googlebotIPNetworkStorageDuration = time.Hour * 24
var googlebotIPNetworkRefreshPeriod = time.Second * 3
var googlebotIPStorage *GooglebotIPStorage
var googlebotIPStorageClient *GooglebotIPStorageClient

type GooglebotIPRecord = string

type GooglebotIPStorage struct {
	records  []GooglebotIPRecord
	networks []*net.IPNet
	mx       sync.Mutex
}

type GooglebotIPStorageClient struct {
	client  *redis.Client
	enabled bool
}

func (c *GooglebotIPStorageClient) Key() string {
	return "googlebot:network"
}

func (c *GooglebotIPStorageClient) IsActive() bool {
	return c.client != nil && c.enabled
}

func (c *GooglebotIPStorageClient) Start() error {
	c.enabled = false

	_, err := c.client.Ping(context.Background()).Result()
	if err == nil {
		c.enabled = true
	}
	return err
}

func (c *GooglebotIPStorageClient) Store(records []GooglebotIPRecord) []GooglebotIPRecord {
	data := strings.Join(records, ", ")
	_, err := c.client.SetEX(context.Background(), c.Key(), data, googlebotIPNetworkStorageDuration).Result()
	if err != nil {
		log.Println("Store Googlebot records,", err.Error())
	}

	return records
}

func (c *GooglebotIPStorageClient) Get() []GooglebotIPRecord {
	records := make([]GooglebotIPRecord, 0)

	data, _ := c.client.Get(context.Background(), c.Key()).Result()
	if data != "" {
		records = strings.Split(data, ", ")
	}

	return records
}

func init() {
	googlebotIPStorage = &GooglebotIPStorage{
		records:  make([]GooglebotIPRecord, 0),
		networks: make([]*net.IPNet, 0),
		mx:       sync.Mutex{},
	}

	googlebotIPStorageClient = &GooglebotIPStorageClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:     "127.0.0.1:6379",
				Password: "",
				DB:       0,
			},
		),
		enabled: false,
	}

	err := googlebotIPStorageClient.Start()
	if err == nil {
		log.Println("Connected to Googlebot ip storage server")
	}
	if err != nil {
		log.Println("Cannot connect to Googlebot ip storage server, falling back to memory storage")
	}

	go getGoogleBotIPs()
}

type GoogleIPPrefix struct {
	IPv4Prefix string `json:"ipv4Prefix"`
	IPv6Prefix string `json:"ipv6Prefix"`
}

type GoogleIPRanges struct {
	Time     CreationTime     `json:"creationTime"`
	Prefixes []GoogleIPPrefix `json:"prefixes"`
}

type CreationTime time.Time

func (c *CreationTime) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	log.Println(value)
	if value == "" || value == "null" {
		return nil
	}

	t, err := time.Parse("2006-01-02T15:04:05", value)
	if err != nil {
		return err
	}
	*c = CreationTime(t)

	return nil
}

func getGoogleBotIPs() {
	var records = make([]GooglebotIPRecord, 0)
	var networks = make([]*net.IPNet, 0)
	var googleIPRanges GoogleIPRanges

	// restoring ip range information
	// from persistent storage to memory storage
	records = googlebotIPStorageClient.Get()
	googlebotIPStorage.records = records

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
		log.Println(err)
		log.Println(googleIPRanges)
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

			networks = append(networks, network)
			if network != nil {
				networks = append(networks, network)
			}
		}

		// putting ip range information to memory storage...
		googlebotIPStorage.mx.Lock()
		googlebotIPStorage.records = records
		googlebotIPStorage.mx.Unlock()

		// ... and updating persistent storage
		googlebotIPStorageClient.Store(records)

		time.Sleep(googlebotIPNetworkRefreshPeriod)
	}
}

func IsGoogleBot(remoteAddr string) bool {
	ip := net.ParseIP(remoteAddr)

	googlebotIPStorage.mx.Lock()
	defer googlebotIPStorage.mx.Unlock()

	for _, network := range googlebotIPStorage.networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
