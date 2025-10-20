package rules

import (
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	. "http-proxy-firewall/lib/firewall/interfaces"
	"http-proxy-firewall/lib/utils"
)

var requestCounters *HostnameRequestCounters
var requestThreshold uint64
var requestCountersSamplingSeconds uint64 = 10
var requestCountersResetPeriod time.Duration

type HostnameRequestCounters struct {
	counters map[string]*HostnameRequestCounter
	mx       sync.RWMutex
}

type HostnameRequestCounter struct {
	createdAt time.Time
	counter   atomic.Uint64
}

var hostnamePenalties *HostnamePenalties
var hostnamePenaltyLifetime time.Duration

type HostnamePenalties struct {
	penalties map[string]*HostnamePenalty
	mx        sync.RWMutex
}

type HostnamePenalty struct {
	expires time.Time
}

func getCounterForHostname(hostname string, now time.Time) uint64 {
	// Try read lock first (fast path for existing counters)
	requestCounters.mx.RLock()
	requestCounter := requestCounters.counters[hostname]
	requestCounters.mx.RUnlock()

	if requestCounter != nil {
		return requestCounter.counter.Add(1)
	}

	// Counter doesn't exist, need write lock
	requestCounters.mx.Lock()
	// Double-check after acquiring write lock
	requestCounter = requestCounters.counters[hostname]
	if requestCounter == nil {
		requestCounter = &HostnameRequestCounter{
			createdAt: now,
		}
		requestCounters.counters[hostname] = requestCounter
	}
	requestCounters.mx.Unlock()

	return requestCounter.counter.Add(1)
}

func setPenaltyForHostname(hostname string, now time.Time) {
	hostnamePenalties.mx.Lock()
	defer hostnamePenalties.mx.Unlock()

	hostnamePenalties.penalties[hostname] = &HostnamePenalty{
		expires: now.Add(hostnamePenaltyLifetime),
	}
}

func hostnameUnderPenalty(hostname string, now time.Time) bool {
	hostnamePenalties.mx.RLock()
	defer hostnamePenalties.mx.RUnlock()

	hostnamePenalty := hostnamePenalties.penalties[hostname]
	return hostnamePenalty != nil && hostnamePenalty.expires.After(now)
}

func init() {
	threshold, err := strconv.ParseUint(utils.GetEnv("DOS_DETECTOR_HOSTNAME_REQUEST_THRESHOLD"), 10, 64)
	if err != nil || threshold == 0 {
		threshold = 100
	}
	requestThreshold = threshold
	requestCountersResetPeriod = time.Duration(requestCountersSamplingSeconds) * time.Second

	lifetime := utils.GetEnv("DOS_DETECTOR_HOSTNAME_PENALTY_LIFETIME")
	if lifetime == "" {
		lifetime = "10m"
	}
	hostnamePenaltyLifetime, err = time.ParseDuration(lifetime)
	if err != nil {
		log.Println("Failed to parse DOS_DETECTOR_HOSTNAME_PENALTY_LIFETIME, using default 10m:", err)
		hostnamePenaltyLifetime = 10 * time.Minute
	}

	requestCounters = &HostnameRequestCounters{
		counters: make(map[string]*HostnameRequestCounter),
		mx:       sync.RWMutex{},
	}

	hostnamePenalties = &HostnamePenalties{
		penalties: make(map[string]*HostnamePenalty),
		mx:        sync.RWMutex{},
	}

	// Reset counters and cleanup old entries periodically
	go func() {
		ticker := time.NewTicker(requestCountersResetPeriod)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()

			// Reset counters atomically
			requestCounters.mx.Lock()
			for hostname, requestCounter := range requestCounters.counters {
				// Remove stale counters (older than 2x the reset period)
				if now.Sub(requestCounter.createdAt) > 2*requestCountersResetPeriod {
					delete(requestCounters.counters, hostname)
				} else {
					// Reset counter using atomic operation
					requestCounter.counter.Store(0)
				}
			}
			requestCounters.mx.Unlock()

			// Cleanup expired penalties
			hostnamePenalties.mx.Lock()
			for hostname, penalty := range hostnamePenalties.penalties {
				if penalty.expires.Before(now) {
					delete(hostnamePenalties.penalties, hostname)
				}
			}
			hostnamePenalties.mx.Unlock()
		}
	}()
}

type DosDetector struct {
}

func isAboveThreshold(hostname string, now time.Time) (bool, uint64, uint64) {
	counter := getCounterForHostname(hostname, now)
	avgPerSecond := counter / requestCountersSamplingSeconds
	return avgPerSecond > requestThreshold, counter, avgPerSecond
}

func (f *DosDetector) Handler(c *fiber.Ctx, remoteIP string, hostname string) FilterResult {
	now := time.Now()

	// Check if hostname is under penalty (returns true if blocked)
	if hostnameUnderPenalty(hostname, now) {
		return PassToNext
	}

	// Check if threshold exceeded
	isAbove, counter, avgPerSecond := isAboveThreshold(hostname, now)
	if !isAbove {
		return BreakLoopResult
	}

	// Threshold exceeded - apply penalty
	log.Println("DoS threshold exceeded:", hostname, "total=", counter, "avg/sec=", avgPerSecond)
	setPenaltyForHostname(hostname, now)
	return PassToNext
}
