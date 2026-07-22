package web

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// limiter est un rate limiter "token bucket" minimal, par IP, en mémoire.
// Suffisant pour éviter un abus trivial de la recherche sans dépendance externe.
type limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    int
	every   time.Duration
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

func newLimiter(rate int, every time.Duration) *limiter {
	l := &limiter{buckets: make(map[string]*bucket), rate: rate, every: every}
	go l.gc()
	return l
}

func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &bucket{tokens: l.rate - 1, lastSeen: now}
		return true
	}

	if refill := int(now.Sub(b.lastSeen) / l.every); refill > 0 {
		b.tokens += refill * l.rate
		if b.tokens > l.rate {
			b.tokens = l.rate
		}
		b.lastSeen = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func (l *limiter) gc() {
	for range time.Tick(5 * time.Minute) {
		l.mu.Lock()
		for ip, b := range l.buckets {
			if time.Since(b.lastSeen) > 10*time.Minute {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
