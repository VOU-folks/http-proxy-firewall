package rules

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func init() {
	ipPenalties = &IpPenalties{}
	ipPenalties.Penalties = make(map[string]*IpPenalty)
	ipPenalties.mx = sync.Mutex{}

	rateLimitZones = &RateLimitZones{}
	rateLimitZones.Zones = make(map[string]*RateLimitZone)

	go func() {
		for {
			ipPenalties.eachPenalty(func(penalty *IpPenalty) {
				penalty.Reset()
			})
			time.Sleep(time.Second * 5)
		}
	}()
}

type IpPenalty struct {
	Time    time.Time
	Counter int8
	Blocked bool
	mx      sync.Mutex
}

func (ipp *IpPenalty) Reset() *IpPenalty {
	ipp.mx.Lock()
	ipp.Time = time.Now()
	ipp.Counter = 0
	ipp.Blocked = false
	ipp.mx.Unlock()

	return ipp
}

func (ipp *IpPenalty) BlockedSince() (bool, time.Time) {
	var blocked bool
	var since time.Time

	ipp.mx.Lock()
	blocked, since = ipp.Blocked, ipp.Time
	ipp.mx.Unlock()

	return blocked, since
}

func (ipp *IpPenalty) Incr() *IpPenalty {
	ipp.mx.Lock()
	ipp.Counter++
	ipp.mx.Unlock()

	return ipp
}

func (ipp *IpPenalty) OverLimit() bool {
	var ov bool

	ipp.mx.Lock()
	ov = ipp.Counter > 2
	ipp.mx.Unlock()

	return ov
}

func (ipp *IpPenalty) Block() *IpPenalty {
	ipp.mx.Lock()
	ipp.Time = time.Now()
	ipp.Blocked = true
	ipp.mx.Unlock()

	return ipp
}

type IpPenalties struct {
	Penalties map[string]*IpPenalty
	mx        sync.Mutex
}

var ipPenalties *IpPenalties

func (ips *IpPenalties) eachPenalty(fn func(penalty *IpPenalty)) {
	ips.mx.Lock()
	for _, penalty := range ips.Penalties {
		fn(penalty)
	}
	ips.mx.Unlock()
}

func (ips *IpPenalties) Get(ip string) *IpPenalty {
	var penalty *IpPenalty

	ips.mx.Lock()
	penalty = ips.Penalties[ip]
	if penalty == nil {
		penalty = &IpPenalty{
			Time:    time.Now(),
			Counter: 0,
			Blocked: false,
			mx:      sync.Mutex{},
		}
	}
	ips.Penalties[ip] = penalty
	ips.mx.Unlock()

	return penalty
}

type TimeCounter struct {
	Time    time.Time
	Counter int
	mx      sync.Mutex
}

func (tc *TimeCounter) Reset() *TimeCounter {
	tc.mx.Lock()
	tc.Time = time.Now()
	tc.Counter = 0
	tc.mx.Unlock()

	return tc
}

func (tc *TimeCounter) Incr() *TimeCounter {
	tc.mx.Lock()
	tc.Time = time.Now()
	tc.Counter++
	tc.mx.Unlock()

	return tc
}

func (tc *TimeCounter) OverLimit() bool {
	var ov bool

	tc.mx.Lock()
	ov = tc.Counter > 5
	tc.mx.Unlock()

	return ov
}

type RateLimitRecord struct {
	IpToTimeCountersMap map[string]*TimeCounter
	mx                  sync.Mutex
}

func (rlr *RateLimitRecord) GetTimeCounter(ip string) *TimeCounter {
	var tc *TimeCounter

	rlr.mx.Lock()
	tc = rlr.IpToTimeCountersMap[ip]
	if tc == nil {
		tc = &TimeCounter{
			Time:    time.Now(),
			Counter: 0,
			mx:      sync.Mutex{},
		}
		rlr.IpToTimeCountersMap[ip] = tc
	}
	rlr.mx.Unlock()

	return tc
}

type RateLimitZone struct {
	Records map[string]*RateLimitRecord
	mx      sync.Mutex
}

func (rlz *RateLimitZone) GetRecord(key string) *RateLimitRecord {
	var rlr *RateLimitRecord

	rlz.mx.Lock()
	rlr = rlz.Records[key]
	if rlr == nil {
		rlr = &RateLimitRecord{
			IpToTimeCountersMap: make(map[string]*TimeCounter),
			mx:                  sync.Mutex{},
		}
		rlz.Records[key] = rlr
	}
	rlz.mx.Unlock()

	return rlr
}

type RateLimitZones struct {
	Zones map[string]*RateLimitZone
	mx    sync.Mutex
}

func (rlzs *RateLimitZones) GetByHost(host string) *RateLimitZone {
	var rlz *RateLimitZone

	rlzs.mx.Lock()
	rlz = rlzs.Zones[host]
	if rlz == nil {
		rlz = &RateLimitZone{
			Records: make(map[string]*RateLimitRecord),
			mx:      sync.Mutex{},
		}
		rlzs.Zones[host] = rlz
	}
	rlzs.mx.Unlock()

	return rlz
}

var rateLimitZones *RateLimitZones

func DoRateLimiting(req *http.Request, res http.ResponseWriter, remoteAddr string) bool {
	if req.Header.Get("Is-Whitelisted") == "true" {
		return false
	}

	now := time.Now()

	ipPenalty := ipPenalties.Get(remoteAddr)

	blocked, _ := ipPenalty.BlockedSince()
	if blocked {
		res.Header().Set("Content-Type", "text/html")
		res.WriteHeader(200)
		_, _ = res.Write([]byte("<meta http-equiv=\"refresh\" content=\"1\">"))
		return true
	}

	url := req.URL.String()
	if !strings.Contains(url, ".php") {
		return false
	}

	if strings.Contains(url, "image") || strings.Contains(url, "img") || strings.Contains(url, "photo") {
		return false
	}

	var host string
	host, _, _ = net.SplitHostPort(req.Host)
	key := req.Method + " " + strings.Split(url, ".php")[0] + ".php"

	rateLimitZone := rateLimitZones.GetByHost(host)
	rateLimitRecord := rateLimitZone.GetRecord(key)
	timeCounter := rateLimitRecord.GetTimeCounter(remoteAddr)

	if now.Sub(timeCounter.Time).Seconds() >= 1 {
		timeCounter.Reset()
	}

	timeCounter.Incr()

	if timeCounter.OverLimit() {
		ipPenalty.Incr()
	}

	if ipPenalty.OverLimit() {
		ipPenalty.Block()

		res.Header().Set("Content-Type", "text/html")
		res.WriteHeader(200)
		_, _ = res.Write([]byte("<meta http-equiv=\"refresh\" content=\"1\">"))
		return true
	}

	return false
}
