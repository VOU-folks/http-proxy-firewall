package rules

import (
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/utils"
)

var requestCounters *HostnameRequestCounters
var requestThreshold uint64
var requestCountersSamplingSeconds uint64 = 10
var requestCountersResetPeriod time.Duration

type HostnameRequestCounters struct {
	counters map[string]*HostnameRequestCounter
	mx       sync.Mutex
}

type HostnameRequestCounter struct {
	time    time.Time
	penalty bool
	counter uint64
}

var hostnamePenalties *HostnamePenalties
var hostnamePenaltyLifetime time.Duration

type HostnamePenalties struct {
	penalties map[string]*HostnamePenalty
	mx        sync.Mutex
}

type HostnamePenalty struct {
	expires time.Time
	penalty bool
}

func getCounterForHostname(hostname string) uint64 {
	requestCounters.mx.Lock()
	defer requestCounters.mx.Unlock()

	requestCounter := requestCounters.counters[hostname]
	if requestCounter == nil {
		requestCounters.counters[hostname] = &HostnameRequestCounter{
			time:    time.Now(),
			counter: 0,
		}
		requestCounter = requestCounters.counters[hostname]
	}
	atomic.AddUint64(&requestCounter.counter, uint64(1))

	return requestCounter.counter
}

func setPenaltyForHostname(hostname string) {
	hostnamePenalties.mx.Lock()
	defer hostnamePenalties.mx.Unlock()

	hostnamePenalties.penalties[hostname] = &HostnamePenalty{
		expires: time.Now().Add(hostnamePenaltyLifetime),
		penalty: true,
	}
}

func hostnameUnderPenalty(hostname string) bool {
	hostnamePenalties.mx.Lock()
	defer hostnamePenalties.mx.Unlock()

	hostnamePenalty := hostnamePenalties.penalties[hostname]
	return hostnamePenalty != nil &&
		hostnamePenalty.penalty == true &&
		hostnamePenalty.expires.After(time.Now())
}

func init() {
	threshold, err := strconv.ParseUint(utils.GetEnv("DOS_DETECTOR_HOSTNAME_REQUEST_THRESHOLD"), 10, 64)
	if err != nil || !(threshold > 0) {
		threshold = 100
	}
	requestThreshold = threshold
	requestCountersResetPeriod, _ = time.ParseDuration(strconv.FormatUint(requestCountersSamplingSeconds, 10) + "s")

	lifetime := utils.GetEnv("DOS_DETECTOR_HOSTNAME_PENALTY_LIFETIME")
	if lifetime == "" {
		lifetime = "10m"
	}
	hostnamePenaltyLifetime, err = time.ParseDuration(lifetime)

	requestCounters = &HostnameRequestCounters{
		counters: make(map[string]*HostnameRequestCounter),
		mx:       sync.Mutex{},
	}

	hostnamePenalties = &HostnamePenalties{
		penalties: make(map[string]*HostnamePenalty),
		mx:        sync.Mutex{},
	}

	go func() {
		for {
			requestCounters.mx.Lock()
			for _, requestCounter := range requestCounters.counters {
				requestCounter.time = time.Now()
				requestCounter.counter = 0
			}
			requestCounters.mx.Unlock()

			time.Sleep(requestCountersResetPeriod)
		}
	}()
}

type DosDetector struct {
}

func isAboveThreshold(hostname string) (bool, uint64, uint64) {
	counter := getCounterForHostname(hostname)
	avgPerSecond := counter / requestCountersSamplingSeconds
	return avgPerSecond > requestThreshold, counter, avgPerSecond
}

func (f *DosDetector) Handler(c *gin.Context, remoteIP string, hostname string) FilterResult {
	if hostnameUnderPenalty(hostname) {
		return PassToNext
	}

	isAbove, counter, avgPerSecond := isAboveThreshold(hostname)
	if !isAbove {
		return BreakLoopResult
	}

	log.Println("isAboveThreshold", hostname, counter, avgPerSecond)
	setPenaltyForHostname(hostname)
	return PassToNext
}
