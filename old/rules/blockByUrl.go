package rules

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ClearanceLoop struct {
	Time              time.Time
	SleepTime         time.Duration
	ResetAfterSeconds float64
}

var clearance = ClearanceLoop{
	SleepTime:         time.Second * 3,
	ResetAfterSeconds: float64(30),
}

func init() {
	urlsTimeCounterMap = &UrlsTimeCounterMap{}
	urlsTimeCounterMap.UrlCounters = make(map[string]*UrlTimeCounter)
	urlsTimeCounterMap.mx = sync.Mutex{}

	go func() {
		for {
			now := time.Now()

			urlsTimeCounterMap.eachCounter(func(key string, utc *UrlTimeCounter) {
				if utc.OverLimit() && !utc.IsBlocked() {
					utc.Block()
					log.Println(key, "blocked")
				}

				if now.Sub(utc.Time).Seconds() > clearance.ResetAfterSeconds {
					utc.Reset()
				}
			})
			time.Sleep(clearance.SleepTime)
		}
	}()
}

var urlsTimeCounterMap *UrlsTimeCounterMap

type UrlTimeCounter struct {
	Time    time.Time
	Counter int
	Blocked bool
	mx      sync.Mutex
}

func (utc *UrlTimeCounter) Reset() *UrlTimeCounter {
	utc.mx.Lock()
	utc.Time = time.Now()
	utc.Counter = 0
	utc.Blocked = false
	utc.mx.Unlock()

	return utc
}

func (utc *UrlTimeCounter) Incr() *UrlTimeCounter {
	utc.mx.Lock()
	utc.Counter++
	utc.mx.Unlock()

	return utc
}

func (utc *UrlTimeCounter) IsBlocked() bool {
	var blocked bool

	utc.mx.Lock()
	blocked = utc.Blocked
	utc.mx.Unlock()

	return blocked
}

func (utc *UrlTimeCounter) Block() *UrlTimeCounter {
	utc.mx.Lock()
	utc.Blocked = true
	utc.mx.Unlock()

	return utc
}

func (utc *UrlTimeCounter) OverLimit() bool {
	var ov bool

	utc.mx.Lock()
	ov = utc.Counter > 10
	utc.mx.Unlock()

	return ov
}

type UrlsTimeCounterMap struct {
	UrlCounters map[string]*UrlTimeCounter
	mx          sync.Mutex
}

func (utcm *UrlsTimeCounterMap) GetTimeCounter(key string) *UrlTimeCounter {
	var tc *UrlTimeCounter

	utcm.mx.Lock()
	tc = utcm.UrlCounters[key]
	if tc == nil {
		tc = &UrlTimeCounter{
			Time:    time.Now(),
			Counter: 0,
			Blocked: false,
			mx:      sync.Mutex{},
		}
		utcm.UrlCounters[key] = tc
	}
	utcm.mx.Unlock()

	return tc
}

func (utcm *UrlsTimeCounterMap) eachCounter(fn func(key string, counter *UrlTimeCounter)) {
	utcm.mx.Lock()
	for key, counter := range utcm.UrlCounters {
		fn(key, counter)
	}
	utcm.mx.Unlock()
}

func DoBlockByUrl(req *http.Request, res http.ResponseWriter) bool {
	if req.Header.Get("Is-Whitelisted") == "true" {
		return false
	}

	url := req.URL.String()
	if !strings.Contains(url, ".php") {
		return false
	}

	host := req.Host
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(req.Host)
	}

	urlParts := strings.Split(url, "?")
	if len(urlParts) == 1 {
		return false
	}

	query := strings.Trim(urlParts[1], " ")
	if query == "" {
		return false
	}

	if strings.Contains(urlParts[0], "image") || strings.Contains(urlParts[0], "img") || strings.Contains(urlParts[0], "photo") {
		return false
	}

	key := host + " " + url
	timeCounter := urlsTimeCounterMap.GetTimeCounter(key)
	if timeCounter.IsBlocked() {
		now := time.Now()
		countdown := int(clearance.ResetAfterSeconds - now.Sub(timeCounter.Time).Seconds())
		if countdown < 0 {
			countdown = 0
		}

		res.Header().Set("Content-Type", "text/html")
		res.WriteHeader(200)
		_, _ = res.Write([]byte(fmt.Sprintf("<meta http-equiv=\"refresh\" content=\"1\">%d", countdown)))
		return true
	}

	timeCounter.Incr()

	return false
}
