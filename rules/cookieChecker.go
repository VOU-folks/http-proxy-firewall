package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jaevor/go-nanoid"
)

var cookieStorageClient *CookieStorageClient

type CookieStorageClient struct {
	client  *redis.Client
	enabled bool
}

func (csc *CookieStorageClient) Key(sid string) string {
	return fmt.Sprintf("cookie:%s", sid)
}

func (csc *CookieStorageClient) IsActive() bool {
	return csc.client != nil && csc.enabled
}

func (csc *CookieStorageClient) Start() error {
	csc.enabled = false

	_, err := csc.client.Ping(context.Background()).Result()
	if err == nil {
		csc.enabled = true
	}
	return err
}

func (csc *CookieStorageClient) Store(record *CookieRecord) *CookieRecord {
	_, err := csc.client.SetEX(context.Background(), csc.Key(record.Sid), record, cookieRecordExtensionTime).Result()
	if err != nil {
		log.Println("Store,", record.Sid, err.Error())
	}

	return record
}

func (csc *CookieStorageClient) Get(sid string) *CookieRecord {
	var cookieRecord *CookieRecord
	var cookieData string

	cookieData, _ = csc.client.Get(context.Background(), csc.Key(sid)).Result()
	if cookieData != "" {
		err := json.Unmarshal([]byte(cookieData), &cookieRecord)
		if err != nil {
			log.Println("Get,", sid, err.Error())
		}

		if cookieRecord != nil {
			cookieRecord.mx = sync.Mutex{}
		}

		return cookieRecord
	}

	return nil
}

func (csc *CookieStorageClient) Delete(sid string) {
	_, _ = csc.client.Del(context.Background(), csc.Key(sid)).Result()
}

func init() {
	cookieStorage = &CookieStorage{
		storage: make(map[string]*CookieRecord),
		mx:      sync.Mutex{},
	}

	cookieStorageClient = &CookieStorageClient{
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
		log.Println("Connected to cookie storage server")
	}
	if err != nil {
		log.Println("Cannot connect to cookie storage server, falling back to memory storage")
	}
}

var cookieRecordExtensionTime = time.Hour * 24
var sidHashSize = 32
var cookieRecordHashSize = 16
var sidCookieLifetime = time.Hour * 24 * 365
var sidCookieName = "pf-x-sid"

var newSidHash, _ = nanoid.Standard(sidHashSize)
var newCookieHash, _ = nanoid.Standard(cookieRecordHashSize)

type CookieRecord struct {
	Created   time.Time `json:"Created" redis:"Create"`
	Expires   time.Time `json:"Expires" redis:"Expires"`
	Sid       string    `json:"Sid" redis:"Sid"`
	Value     string    `json:"Value" redis:"Value"`
	IP        string    `json:"IP" redis:"IP"`
	UserAgent string    `json:"UserAgent" redis:"UserAgent"`
	Domain    string    `json:"Domain" redis:"Domain"`
	mx        sync.Mutex
}

func (cr *CookieRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(cr)
}

func (cr *CookieRecord) Refresh() {
	cr.mx.Lock()
	cr.Expires = time.Now().Add(cookieRecordExtensionTime)
	cr.mx.Unlock()

	if cookieStorageClient.IsActive() {
		cookieStorageClient.Store(cr)
	}
}

func (cr *CookieRecord) Outdated() bool {
	cr.mx.Lock()
	outdated := time.Now().Sub(cr.Expires).Seconds() > cookieRecordExtensionTime.Seconds()
	cr.mx.Unlock()

	if cookieStorageClient.IsActive() {
		if outdated {
			go cookieStorageClient.Delete(cr.Sid)
		}
	}

	return outdated
}

func (cr *CookieRecord) IsAllowed(sid string, req *http.Request, remoteAddr string, domain string) bool {
	cookie, _ := req.Cookie(sid)

	// allowed := cookie != nil &&
	//  cookie.Domain == cr.Domain &&
	// 	cookie.Value == cr.Value &&
	// 	cr.IP == remoteAddr &&
	// 	cr.UserAgent == req.Header.Get("User-Agent")

	// var ipParts = strings.Split(cr.IP, ".")
	// var subIp = fmt.Sprintf("%d.%d.%d.", ipParts[0], ipParts[1], ipParts[2])
	// var remoteAddrParts = strings.Split(remoteAddr, ".")
	// var subRemoteAddr = fmt.Sprintf("%d.%d.%d.", remoteAddrParts[0], remoteAddrParts[1], remoteAddrParts[2])

	allowed := cookie != nil &&
		cr.Domain == domain &&
		cookie.Value == cr.Value &&
		cr.UserAgent == req.Header.Get("User-Agent")

	return allowed
}

func NewCookieRecord(sid string, req *http.Request, remoteAddr string, domain string) *CookieRecord {
	cookie := &CookieRecord{
		Created:   time.Now(),
		Expires:   time.Now().Add(cookieRecordExtensionTime),
		Sid:       sid,
		Value:     newCookieHash(),
		IP:        remoteAddr,
		UserAgent: req.Header.Get("User-Agent"),
		Domain:    domain,
		mx:        sync.Mutex{},
	}

	return cookie
}

type CookieStorage struct {
	storage map[string]*CookieRecord
	mx      sync.Mutex
}

func (cs *CookieStorage) GetCookieRecord(sid string) *CookieRecord {
	if cookieStorageClient.IsActive() {
		return cookieStorageClient.Get(sid)
	}

	cs.mx.Lock()
	cookieRecord := cs.storage[sid]
	cs.mx.Unlock()

	return cookieRecord
}

func (cs *CookieStorage) CreateCookieRecord(sid string, req *http.Request, remoteAddr string, domain string) *CookieRecord {
	cookieRecord := NewCookieRecord(sid, req, remoteAddr, domain)

	if cookieStorageClient.IsActive() {
		return cookieStorageClient.Store(cookieRecord)
	}

	cs.mx.Lock()
	cs.storage[sid] = cookieRecord
	cs.mx.Unlock()

	return cookieRecord
}

func (cs *CookieStorage) DeleteCookieRecord(sid string) {
	if cookieStorageClient.IsActive() {
		cookieStorageClient.Delete(sid)
		return
	}

	cs.mx.Lock()
	delete(cs.storage, sid)
	cs.mx.Unlock()
}

var cookieStorage *CookieStorage

func showAuthPage(req *http.Request, res http.ResponseWriter, cookie *CookieRecord) {
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(200)
	_, _ = res.Write([]byte("<meta http-equiv=\"refresh\" content=\"0\">"))
}

func sendCookieRecord(req *http.Request, res http.ResponseWriter, cookieRecord *CookieRecord) *http.Cookie {
	cookie := &http.Cookie{}
	cookie.Name = cookieRecord.Sid
	cookie.Value = cookieRecord.Value
	cookie.Expires = cookieRecord.Expires
	cookie.Path = "/"
	cookie.Domain = cookieRecord.Domain
	cookie.HttpOnly = false
	http.SetCookie(res, cookie)

	return cookie
}

func retractCookieRecord(req *http.Request, res http.ResponseWriter, cookieRecord *CookieRecord) {
	if cookieRecord == nil {
		return
	}

	cookie := &http.Cookie{}
	cookie.Name = cookieRecord.Sid
	cookie.Value = cookieRecord.Value
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	cookie.Path = "/"
	cookie.Domain = cookieRecord.Domain
	cookie.HttpOnly = false
	http.SetCookie(res, cookie)
}

func retractSidCookie(req *http.Request, res http.ResponseWriter, domain string) *http.Cookie {
	cookie := &http.Cookie{}
	cookie.Name = sidCookieName
	cookie.Value = ""
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	cookie.Path = "/"
	cookie.Domain = domain
	cookie.HttpOnly = false
	http.SetCookie(res, cookie)

	return cookie
}

func sendNewSidCookie(req *http.Request, res http.ResponseWriter, domain string) *http.Cookie {
	cookie := &http.Cookie{}
	cookie.Name = sidCookieName
	cookie.Value = newSidHash()
	cookie.Expires = time.Now().Add(sidCookieLifetime)
	cookie.Path = "/"
	cookie.Domain = domain
	cookie.HttpOnly = false
	http.SetCookie(res, cookie)

	return cookie
}

func ReauthorizeRequest(req *http.Request, res http.ResponseWriter, remoteAddr string, domain string) {
	sidCookie := sendNewSidCookie(req, res, domain)

	cookieRecord := cookieStorage.CreateCookieRecord(sidCookie.Value, req, remoteAddr, domain)
	sendCookieRecord(req, res, cookieRecord)

	showAuthPage(req, res, cookieRecord)
}

func RejectSidCookie(req *http.Request) {
	sidCookie, _ := req.Cookie(sidCookieName)
	if sidCookie != nil {
		cookieStorage.DeleteCookieRecord(sidCookie.Value)
	}
}

func DeleteOldCookieRecords(req *http.Request, res http.ResponseWriter, keepCookieName string, domain string) {
	for _, cookie := range req.Cookies() {
		if cookie.Name != keepCookieName &&
			len(cookie.Name) > sidHashSize && len(cookie.Value) > cookieRecordHashSize {
			cookieStorage.DeleteCookieRecord(cookie.Name)

			cookie.Value = ""
			cookie.MaxAge = -1
			cookie.Expires = time.Unix(0, 0)
			cookie.Path = "/"
			cookie.Domain = domain
			cookie.HttpOnly = false
			http.SetCookie(res, cookie)
		}
	}
}

func AuthorizeByCookie(req *http.Request, res http.ResponseWriter, remoteAddr string) bool {
	if req.Header.Get("Is-Whitelisted") == "true" {
		return true
	}

	domain := req.Host
	host, _, _ := net.SplitHostPort(req.Host)
	if host != "" {
		domain = host
	}

	var cookieRecord *CookieRecord

	sidCookie, _ := req.Cookie(sidCookieName)
	if sidCookie == nil {
		ReauthorizeRequest(req, res, remoteAddr, domain)
		return false
	}

	DeleteOldCookieRecords(req, res, sidCookie.Value, domain)

	cookieRecord = cookieStorage.GetCookieRecord(sidCookie.Value)

	if cookieRecord == nil ||
		cookieRecord.Outdated() ||
		!cookieRecord.IsAllowed(sidCookie.Value, req, remoteAddr, domain) {

		retractCookieRecord(req, res, cookieRecord)
		cookieStorage.DeleteCookieRecord(sidCookie.Value)

		ReauthorizeRequest(req, res, remoteAddr, domain)

		return false
	}

	cookieRecord.Refresh()

	return true
}
