package country

import (
	"context"
	"encoding/json"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"http-proxy-firewall/lib/utils"
)

var ipToCountryStorageDuration = time.Hour * 24
var ipToCountryStorageShortDuration = time.Hour
var redisTimeout = time.Second * 5

var ipToCountryStorage *IpToCountryStorage
var ipToCountryStorageClient *IpToCountryStorageClient

type IpToCountry struct {
	Created time.Time `json:"created" redis:"created"`
	Expires time.Time `json:"expires" redis:"expires"`
	IP      string    `json:"ip" redis:"ip"`
	Country string    `json:"country" redis:"country"`
}

func (ipc *IpToCountry) MarshalBinary() ([]byte, error) {
	return json.Marshal(ipc)
}

type IpToCountryStorage struct {
	storage map[string]IpToCountry
	mx      sync.RWMutex
}

func (s *IpToCountryStorage) Get(key string) (IpToCountry, bool) {
	s.mx.RLock()
	result, exists := s.storage[key]
	s.mx.RUnlock()

	return result, exists
}

func (s *IpToCountryStorage) Store(ipToCountry IpToCountry) {
	s.mx.Lock()
	s.storage[ipToCountry.IP] = ipToCountry
	s.mx.Unlock()
}

type IpToCountryStorageClient struct {
	client    *redis.Client
	enabled   bool
	connected bool
	mx        sync.RWMutex
}

func EnableRedisClient(enable bool) {
	ipToCountryStorageClient.mx.Lock()
	ipToCountryStorageClient.enabled = enable
	ipToCountryStorageClient.mx.Unlock()
}

func (c *IpToCountryStorageClient) StorageKey() string {
	return "IPTOCOUNTRY"
}

func (c *IpToCountryStorageClient) Key(entry string) string {
	return entry
}

func (c *IpToCountryStorageClient) KeyFromIpToCountry(ipToCountry IpToCountry) string {
	return c.Key(ipToCountry.IP)
}

func (c *IpToCountryStorageClient) IsActive() bool {
	c.mx.RLock()
	result := c.enabled && c.client != nil && c.connected
	c.mx.RUnlock()
	return result
}

func (c *IpToCountryStorageClient) Start() {
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
			}

			time.Sleep(time.Second * 10)
		}
	}()
}

func (c *IpToCountryStorageClient) Store(ipToCountry IpToCountry) {
	data, _ := json.Marshal(ipToCountry)
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	_, err := c.client.HSet(ctx, c.StorageKey(), c.KeyFromIpToCountry(ipToCountry), data).Result()
	if err != nil {
		log.Println("IpToCountryStorageClient.Store", ipToCountry, err.Error())
	}
}

func (c *IpToCountryStorageClient) Get(ip string) (IpToCountry, bool) {
	var ipToCountry IpToCountry

	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	data, _ := c.client.HGet(ctx, c.StorageKey(), c.Key(ip)).Result()
	if data != "" {
		err := json.Unmarshal([]byte(data), &ipToCountry)
		if err != nil {
			log.Println("IpToCountryStorageClient.Get", ip, err.Error())
			return ipToCountry, false
		}

		return ipToCountry, true
	}

	return ipToCountry, false
}

func init() {
	ipToCountryStorage = &IpToCountryStorage{
		storage: make(map[string]IpToCountry),
		mx:      sync.RWMutex{},
	}

	// Calculate optimal pool size: at least 10, or 4x CPU cores
	poolSize := runtime.NumCPU() * 4
	if poolSize < 10 {
		poolSize = 10
	}

	ipToCountryStorageClient = &IpToCountryStorageClient{
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

	ipToCountryStorageClient.Start()
}

func ResolveCountryByIP(remoteAddr string) string {
	now := time.Now()

	// get from memory storage
	ipToCountry, exists := ipToCountryStorage.Get(remoteAddr)
	if exists {
		if !ipToCountry.Expires.Before(now) {
			return ipToCountry.Country
		}
	}

	// trying to get from external storage
	if ipToCountryStorageClient.IsActive() {
		ipToCountry, exists = ipToCountryStorageClient.Get(remoteAddr)
		if exists {
			if !ipToCountry.Expires.Before(now) {
				ipToCountryStorage.Store(ipToCountry) // storing to memory storage
				return ipToCountry.Country
			}
		}
	}

	// default information with short lifetime
	// to not request external IP service too often
	ipToCountry = IpToCountry{
		Created: now,
		Expires: now.Add(ipToCountryStorageShortDuration),
		Country: "",
		IP:      remoteAddr,
	}

	// requesting IP service to get information
	response, found := utils.ResolveUsingMaxMindAPI(remoteAddr)
	if found {
		// prolonging lifetime of IP information
		ipToCountry.Expires = now.Add(ipToCountryStorageDuration)
		ipToCountry.Country = response.Country
	}

	ipToCountryStorage.Store(ipToCountry) // storing to memory storage
	if ipToCountryStorageClient.IsActive() {
		go ipToCountryStorageClient.Store(ipToCountry) // storing to external storage
	}

	return ipToCountry.Country
}
