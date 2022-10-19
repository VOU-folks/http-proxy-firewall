package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var ipStorage *IpStorage
var ipFilterClient *IpFilterClient
var ipRecordValidityDuration = time.Hour * 14 * 24

type IpRecord struct {
	Created time.Time `json:"Created" redis:"Create"`
	Expires time.Time `json:"Expires" redis:"Expires"`
	IP      string    `json:"IP" redis:"IP"`
	mx      sync.Mutex
}

type IpStorage struct {
	storage map[string]*IpRecord
	mx      sync.Mutex
}

type IpFilterClient struct {
	client  *redis.Client
	enabled bool
}

func (ifc *IpFilterClient) Key(ip string) string {
	return fmt.Sprintf("ip:%s", ip)
}

func (ifc *IpFilterClient) IsActive() bool {
	return ifc.client != nil && ifc.enabled
}

func (ifc *IpFilterClient) Start() error {
	ifc.enabled = false

	_, err := ifc.client.Ping(context.Background()).Result()
	if err == nil {
		ifc.enabled = true
	}
	return err
}

func (ifc *IpFilterClient) Store(ipRecord *IpRecord) *IpRecord {
	_, err := ifc.client.SetEX(context.Background(), ifc.Key(ipRecord.IP), ipRecord, ipRecordValidityDuration).Result()
	if err != nil {
		log.Println("Store,", ipRecord.IP, err.Error())
	}

	return ipRecord
}

func (ifc *IpFilterClient) Get(ip string) *IpRecord {
	var ipRecord *IpRecord
	var ipData string

	ipData, _ = ifc.client.Get(context.Background(), ifc.Key(ip)).Result()
	if ipData != "" {
		err := json.Unmarshal([]byte(ipData), &ipRecord)
		if err != nil {
			log.Println("Get,", ip, err.Error())
		}

		if ipRecord != nil {
			ipRecord.mx = sync.Mutex{}
		}

		return ipRecord
	}

	return nil
}

func (ifc *IpFilterClient) Delete(ip string) {
	_, _ = ifc.client.Del(context.Background(), ifc.Key(ip)).Result()
}

func init() {
	ipStorage = &IpStorage{
		storage: make(map[string]*IpRecord),
		mx:      sync.Mutex{},
	}

	ipFilterClient = &IpFilterClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:     "127.0.0.1:6379",
				Password: "",
				DB:       0,
			},
		),
		enabled: false,
	}

	err := cookieStorageClient.Start()
	if err == nil {
		log.Println("Connected to ip storage server")
	}
	if err != nil {
		log.Println("Cannot connect to ip storage server, falling back to memory storage")
	}
}

func AllowIP(remoteAddress string) bool {
	ip := net.ParseIP(remoteAddress)
	if ip.IsLoopback() {
		return true
	}

	if IsGoogleBot(remoteAddress) {
		return true
	}

	return false
}
