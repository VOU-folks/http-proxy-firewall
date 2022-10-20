package country

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"http-proxy-firewall/lib/utils"
)

var ipToCountryStorageDuration = time.Hour * 15 * 24
var ipToCountryStorageShortDuration = time.Hour * 24

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
	storage map[string]*IpToCountry
	mx      sync.Mutex
}

func (s *IpToCountryStorage) Get(key string) *IpToCountry {
	s.mx.Lock()
	result := s.storage[key]
	s.mx.Unlock()

	return result
}

func (s *IpToCountryStorage) Store(ipToCountry *IpToCountry) {
	if ipToCountry == nil {
		return
	}

	s.mx.Lock()
	s.storage[ipToCountry.IP] = ipToCountry
	s.mx.Unlock()
}

type IpToCountryStorageClient struct {
	client  *redis.Client
	enabled bool
	mx      sync.Mutex
}

func (c *IpToCountryStorageClient) StorageKey() string {
	return "IPTOCOUNTRY"
}

func (c *IpToCountryStorageClient) Key(entry string) string {
	return entry
}

func (c *IpToCountryStorageClient) KeyFromIpToCountry(ipToCountry *IpToCountry) string {
	return c.Key(ipToCountry.IP)
}

func (c *IpToCountryStorageClient) IsActive() bool {
	c.mx.Lock()
	result := c.client != nil && c.enabled
	c.mx.Unlock()

	return result
}

func (c *IpToCountryStorageClient) Start() {
	c.enabled = false

	go func() {
		for {
			_, err := c.client.Ping(context.Background()).Result()
			c.mx.Lock()
			c.enabled = err == nil
			c.mx.Unlock()

			time.Sleep(time.Minute)
		}
	}()
}

func (c *IpToCountryStorageClient) Store(ipToCountry *IpToCountry) {
	if ipToCountry == nil {
		return
	}

	data, _ := json.Marshal(ipToCountry)
	_, err := c.client.HSet(context.Background(), c.StorageKey(), c.KeyFromIpToCountry(ipToCountry), data).Result()
	if err != nil {
		log.Println("IpToCountryStorageClient.Store", ipToCountry, err.Error())
	}
}

func (c *IpToCountryStorageClient) Get(ip string) *IpToCountry {
	var ipToCountry *IpToCountry

	data, _ := c.client.HGet(context.Background(), c.StorageKey(), c.Key(ip)).Result()
	if data != "" {
		err := json.Unmarshal([]byte(data), &ipToCountry)
		if err != nil {
			log.Println("IpToCountryStorageClient.Get", ip, err.Error())
			return nil
		}
	}

	return ipToCountry
}

func init() {
	ipToCountryStorage = &IpToCountryStorage{
		storage: make(map[string]*IpToCountry, 0),
		mx:      sync.Mutex{},
	}

	ipToCountryStorageClient = &IpToCountryStorageClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:     "redis:6379",
				Password: "",
				DB:       0,
			},
		),
		enabled: false,
		mx:      sync.Mutex{},
	}

	ipToCountryStorageClient.Start()
}

func ResolveCountryByIP(remoteAddr string) string {
	// get from memory storage
	ipToCountry := ipToCountryStorage.Get(remoteAddr)
	if ipToCountry != nil {
		if !ipToCountry.Expires.Before(time.Now()) {
			return ipToCountry.Country
		}
	}

	// if it was not in memory storage it shall be nil
	if ipToCountry == nil {
		// trying to get from external storage
		if ipToCountryStorageClient.IsActive() {
			ipToCountry = ipToCountryStorageClient.Get(remoteAddr)
			if ipToCountry != nil {
				if !ipToCountry.Expires.Before(time.Now()) {
					ipToCountryStorage.Store(ipToCountry) // storing to memory storage
					return ipToCountry.Country
				}
			}
		}
	}

	// default information with short lifetime
	// to not request external IP service too often
	ipToCountry = &IpToCountry{
		Created: time.Now(),
		Expires: time.Now().Add(ipToCountryStorageShortDuration),
		Country: "",
		IP:      remoteAddr,
	}

	// requesting IP service to get information
	response := utils.ResolveUsingMaxMindAPI(remoteAddr)
	if response != nil {
		// prolonging lifetime of iIP information
		ipToCountry.Expires = time.Now().Add(ipToCountryStorageDuration)
		ipToCountry.Country = response.Country
	}

	ipToCountryStorage.Store(ipToCountry) // storing to memory storage
	if ipToCountryStorageClient.IsActive() {
		go ipToCountryStorageClient.Store(ipToCountry) // storing to external storage
	}

	return ipToCountry.Country
}
