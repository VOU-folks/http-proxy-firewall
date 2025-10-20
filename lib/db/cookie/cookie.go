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

var cookieStorageDuration = time.Hour * 24
var redisTimeout = time.Second * 5

var cookieStorage *CookieStorage
var cookieStorageClient *CookieStorageClient

var cookieAccessJournal *CookieAccessJournal

type CookieAccessJournal struct {
	records map[string]time.Time
	mx      sync.RWMutex
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
	now := time.Now()
	expirationThreshold := cookieStorageDuration.Seconds()

	// Collect expired SIDs while holding read lock
	j.mx.RLock()
	for sid, accessTime := range j.records {
		if now.Sub(accessTime).Seconds() >= expirationThreshold {
			sids = append(sids, sid)
		}
	}
	j.mx.RUnlock()

	// Delete expired cookies
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

func (cr *CookieRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(cr)
}

type CookieStorage struct {
	storage map[string]*CookieRecord
	mx      sync.RWMutex
}

func (cs *CookieStorage) Get(key string) *CookieRecord {
	cs.mx.RLock()
	result := cs.storage[key]
	cs.mx.RUnlock()

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
	client    *redis.Client
	enabled   bool
	connected bool
	mx        sync.RWMutex
}

func EnableRedisClient(enable bool) {
	cookieStorageClient.enabled = enable
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
	c.mx.RLock()
	result := c.client != nil && c.enabled
	c.mx.RUnlock()

	return result
}

func (c *CookieStorageClient) Start() {
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

			time.Sleep(time.Second * 5)
		}
	}()
}

func (c *CookieStorageClient) Store(cookieRecord *CookieRecord) {
	if cookieRecord == nil {
		return
	}

	data, _ := json.Marshal(cookieRecord)
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	_, err := c.client.SetEX(ctx, c.KeyFromCookieRecord(cookieRecord), data, cookieStorageDuration).Result()
	if err != nil {
		log.Println("CookieStorageClient.Store", cookieRecord, err.Error())
	}
}

func (c *CookieStorageClient) Get(sid string) *CookieRecord {
	var cookieRecord *CookieRecord

	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	data, _ := c.client.Get(ctx, c.Key(sid)).Result()
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
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	_, _ = c.client.Del(ctx, c.Key(sid)).Result()
}

func init() {
	cookieAccessJournal = &CookieAccessJournal{
		records: make(map[string]time.Time),
		mx:      sync.RWMutex{},
	}

	cookieStorage = &CookieStorage{
		storage: make(map[string]*CookieRecord),
		mx:      sync.RWMutex{},
	}

	cookieAccessJournal.Start(cookieStorage)

	// Calculate optimal pool size: at least 10, or 4x CPU cores
	poolSize := runtime.NumCPU() * 4
	if poolSize < 10 {
		poolSize = 10
	}

	cookieStorageClient = &CookieStorageClient{
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

	cookieStorageClient.Start()
}

func GetCookieRecordBySid(sid string) *CookieRecord {
	now := time.Now()

	// get from memory storage
	cookieRecord := cookieStorage.Get(sid)
	if cookieRecord != nil {
		if !cookieRecord.Expires.Before(now) {
			// Only use goroutine for journal access update (non-critical path)
			go cookieAccessJournal.Accessed(sid)
			return cookieRecord
		}
		// Expired - delete synchronously (fast operations)
		cookieStorage.Delete(sid)
		cookieAccessJournal.Delete(sid)
		if cookieStorageClient.IsActive() {
			go cookieStorageClient.Delete(sid) // Redis delete can be async
		}
		return nil
	}

	// Try external storage (Redis)
	if cookieStorageClient.IsActive() {
		cookieRecord = cookieStorageClient.Get(sid)
		if cookieRecord != nil {
			if !cookieRecord.Expires.Before(now) {
				// Valid cookie from Redis - store in memory cache
				cookieStorage.Store(cookieRecord)
				return cookieRecord
			}
			// Expired - delete from Redis
			go cookieStorageClient.Delete(sid)
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
