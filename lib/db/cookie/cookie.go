package cookie

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jaevor/go-nanoid"
)

var cookieStorageDuration = time.Hour * 24
var cookieStorageCapacity = 100000

var cookieStorage *CookieStorage
var cookieStorageClient *CookieStorageClient

type CookieRecord struct {
	Sid     string    `json:"sid" redis:"sid"`
	Nonce   string    `json:"nonce" redis:"nonce"`
	Expires time.Time `json:"expires" redis:"expires"`
}

func (cr *CookieRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(cr)
}

type CookieStorage struct {
	storage map[string]*CookieRecord
	mx      sync.Mutex
}

func (cs *CookieStorage) Get(key string) *CookieRecord {
	cs.mx.Lock()
	result := cs.storage[key]
	cs.mx.Unlock()

	return result
}

func (cs *CookieStorage) Store(cookieRecord *CookieRecord) {
	if cookieRecord == nil {
		return
	}

	cs.mx.Lock()
	cs.storage[cookieRecord.Sid] = cookieRecord
	cs.mx.Unlock()
}

func (cs *CookieStorage) Delete(key string) {
	cs.mx.Lock()
	delete(cs.storage, key)
	cs.mx.Unlock()
}

type CookieStorageClient struct {
	client  *redis.Client
	enabled bool
	mx      sync.Mutex
}

func (c *CookieStorageClient) StorageKey() string {
	return "COOKIES"
}

func (c *CookieStorageClient) Key(entry string) string {
	return c.StorageKey() + ":" + entry
}

func (c *CookieStorageClient) KeyFromCookieRecord(cookieRecord *CookieRecord) string {
	return c.StorageKey() + ":" + cookieRecord.Sid
}

func (c *CookieStorageClient) IsActive() bool {
	c.mx.Lock()
	result := c.client != nil && c.enabled
	c.mx.Unlock()

	return result
}

func (c *CookieStorageClient) Start() {
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

func (c *CookieStorageClient) Store(cookieRecord *CookieRecord) {
	if cookieRecord == nil {
		return
	}

	data, _ := json.Marshal(cookieRecord)
	_, err := c.client.SetEX(context.Background(), c.KeyFromCookieRecord(cookieRecord), data, cookieStorageDuration).Result()
	if err != nil {
		log.Println("CookieStorageClient.Store", cookieRecord, err.Error())
	}
}

func (c *CookieStorageClient) Get(sid string) *CookieRecord {
	var cookieRecord *CookieRecord

	data, _ := c.client.Get(context.Background(), c.Key(sid)).Result()
	if data != "" {
		err := json.Unmarshal([]byte(data), &cookieRecord)
		if err != nil {
			log.Println("CookieStorageClient.Get", sid, err.Error())
			return nil
		}
	}

	return cookieRecord
}

func (c *CookieStorageClient) Delete(sid string) {
	_, _ = c.client.Del(context.Background(), c.Key(sid)).Result()
}

func init() {
	cookieStorage = &CookieStorage{
		storage: make(map[string]*CookieRecord, 0),
		mx:      sync.Mutex{},
	}

	cookieStorageClient = &CookieStorageClient{
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

	cookieStorageClient.Start()
}

func GetCookieRecordBySid(sid string) *CookieRecord {
	// get from memory storage
	cookieRecord := cookieStorage.Get(sid)
	if cookieRecord != nil {
		if !cookieRecord.Expires.Before(time.Now()) {
			return cookieRecord
		}
		cookieStorage.Delete(sid)
		if cookieStorageClient.IsActive() {
			go cookieStorageClient.Delete(sid)
		}
	}

	// if it was not in memory storage it shall be nil
	if cookieRecord == nil {
		// trying to get from external storage
		if cookieStorageClient.IsActive() {
			cookieRecord = cookieStorageClient.Get(sid)
			if cookieRecord != nil {
				if !cookieRecord.Expires.Before(time.Now()) {
					cookieStorage.Store(cookieRecord) // storing to memory storage
					return cookieRecord
				}
				go cookieStorageClient.Delete(sid)
			}
		}
	}

	return nil
}

var makeNonce, _ = nanoid.Standard(32)

func makeSid(nonce string, remoteAddr string, domain string, userAgent string) string {
	hasher := sha512.New()
	hasher.Write([]byte(domain + ":" + nonce + ":" + userAgent))
	sid := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return sid
}

func NewCookieRecord(remoteAddr string, domain string, userAgent string) *CookieRecord {
	nonce := makeNonce()
	sid := makeSid(nonce, remoteAddr, domain, userAgent)

	cookie := &CookieRecord{
		Nonce:   nonce,
		Sid:     sid,
		Expires: time.Now().Add(cookieStorageDuration),
	}

	return cookie
}

func StoreCookieRecord(cookieRecord *CookieRecord) {
	cookieStorage.Store(cookieRecord)
	if cookieStorageClient.IsActive() {
		go cookieStorageClient.Store(cookieRecord)
	}
}

func ValidateSid(providedSid string, remoteAddr string, domain string, userAgent string) bool {
	cookieRecord := GetCookieRecordBySid(providedSid)
	if cookieRecord == nil {
		return false
	}

	generatedSid := makeSid(
		cookieRecord.Nonce,
		remoteAddr,
		domain,
		userAgent,
	)

	return providedSid == generatedSid
}
