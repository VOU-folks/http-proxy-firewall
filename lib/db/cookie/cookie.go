package cookie

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jaevor/go-nanoid"
)

var cookieStorageDuration = time.Hour

var cookieStorage *CookieStorage
var cookieStorageClient *CookieStorageClient

var cookieAccessJournal *CookieAccessJournal

type CookieAccessJournal struct {
	records map[string]time.Time
	mx      sync.Mutex
}

func (j *CookieAccessJournal) Accessed(sid string) {
	j.mx.Lock()
	j.records[sid] = time.Now()
	j.mx.Unlock()
}

func (j *CookieAccessJournal) Delete(sid string) {
	j.mx.Lock()
	delete(j.records, sid)
	j.mx.Unlock()
}

func (j *CookieAccessJournal) CleanUnusedCookies(cookieStorage *CookieStorage) {
	var sids []string
	var journal map[string]time.Time

	j.mx.Lock()
	journal = j.records
	j.mx.Unlock()

	for sid, accessTime := range journal {
		if time.Now().Sub(accessTime).Seconds() < cookieStorageDuration.Seconds() {
			sids = append(sids, sid)
		}
	}

	if len(sids) > 0 {
		for _, sid := range sids {
			j.Delete(sid)
			cookieStorage.Delete(sid)
		}
	}
}

func (j *CookieAccessJournal) Start(cookieStorage *CookieStorage) {
	go func() {
		for {
			j.CleanUnusedCookies(cookieStorage)

			time.Sleep(time.Hour)
		}
	}()
}

type CookieRecord struct {
	Sid     string    `json:"sid" redis:"sid"`
	Nonce   string    `json:"nonce" redis:"nonce"`
	Expires time.Time `json:"expires" redis:"expires"`
}

func (cr CookieRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(cr)
}

type CookieStorage struct {
	storage map[string]CookieRecord
	mx      sync.Mutex
}

func (cs *CookieStorage) Get(key string) (CookieRecord, bool) {
	cs.mx.Lock()
	result, exists := cs.storage[key]
	cs.mx.Unlock()

	return result, exists
}

func (cs *CookieStorage) Store(cookieRecord CookieRecord) {
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

func (c *CookieStorageClient) KeyFromCookieRecord(cookieRecord CookieRecord) string {
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

func (c *CookieStorageClient) Store(cookieRecord CookieRecord) {
	data, _ := json.Marshal(cookieRecord)
	_, err := c.client.SetEX(context.Background(), c.KeyFromCookieRecord(cookieRecord), data, cookieStorageDuration).Result()
	if err != nil {
		log.Println("CookieStorageClient.Store", cookieRecord, err.Error())
	}
}

func (c *CookieStorageClient) Get(sid string) (CookieRecord, bool) {
	var cookieRecord CookieRecord

	data, _ := c.client.Get(context.Background(), c.Key(sid)).Result()
	if data != "" {
		err := json.Unmarshal([]byte(data), &cookieRecord)
		if err != nil {
			log.Println("CookieStorageClient.Get", sid, err.Error())
			return cookieRecord, false
		}

		return cookieRecord, true
	}

	return cookieRecord, false
}

func (c *CookieStorageClient) Delete(sid string) {
	_, _ = c.client.Del(context.Background(), c.Key(sid)).Result()
}

func init() {
	cookieAccessJournal = &CookieAccessJournal{
		records: make(map[string]time.Time),
		mx:      sync.Mutex{},
	}

	cookieStorage = &CookieStorage{
		storage: make(map[string]CookieRecord),
		mx:      sync.Mutex{},
	}

	cookieAccessJournal.Start(cookieStorage)

	cookieStorageClient = &CookieStorageClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:        "redis:6379",
				Password:    "",
				DB:          0,
				PoolSize:    runtime.NumCPU(),
				PoolTimeout: time.Second * 10,
			},
		),
		enabled: false,
		mx:      sync.Mutex{},
	}

	cookieStorageClient.Start()
}

func GetCookieRecordBySid(sid string) (CookieRecord, bool) {
	// get from memory storage
	cookieRecord, exists := cookieStorage.Get(sid)
	if exists {
		if !cookieRecord.Expires.Before(time.Now()) {
			cookieAccessJournal.Accessed(sid)
			return cookieRecord, true
		}
		cookieStorage.Delete(sid)
		cookieAccessJournal.Delete(sid)
		if cookieStorageClient.IsActive() {
			go cookieStorageClient.Delete(sid)
		}
	}

	// trying to get from external storage
	if cookieStorageClient.IsActive() {
		cookieRecord, exists = cookieStorageClient.Get(sid)
		if exists {
			if !cookieRecord.Expires.Before(time.Now()) {
				cookieStorage.Store(cookieRecord) // storing to memory storage
				return cookieRecord, true
			}
			go cookieStorageClient.Delete(sid)
		}
	}

	return cookieRecord, false
}

var makeNonce, _ = nanoid.Standard(32)

func makeSid(nonce string, remoteAddr string, domain string, userAgent string) string {
	hasher := sha512.New()
	hasher.Write([]byte(domain + ":" + nonce + ":" + userAgent))
	sid := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return sid
}

func NewCookieRecord(remoteAddr string, domain string, userAgent string) CookieRecord {
	nonce := makeNonce()
	sid := makeSid(nonce, remoteAddr, domain, userAgent)

	cookie := CookieRecord{
		Nonce:   nonce,
		Sid:     sid,
		Expires: time.Now().Add(cookieStorageDuration),
	}

	return cookie
}

func StoreCookieRecord(cookieRecord CookieRecord) {
	cookieStorage.Store(cookieRecord)
	if cookieStorageClient.IsActive() {
		go cookieStorageClient.Store(cookieRecord)
	}
}

func ValidateSid(providedSid string, remoteAddr string, domain string, userAgent string) bool {
	cookieRecord, exists := GetCookieRecordBySid(providedSid)
	if !exists {
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
