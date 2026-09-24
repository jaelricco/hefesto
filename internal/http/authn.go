package http

import (
	"context"
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/jaelricco/hefesto/internal/auth"
)

type principalKey struct{}

// principalFrom returns the authenticated principal. Only called behind
// requireAuth, which guarantees one is present.
func principalFrom(ctx context.Context) auth.Principal {
	p, _ := ctx.Value(principalKey{}).(auth.Principal)
	return p
}

// requireAuth admits requests with a valid bearer access token.
func requireAuth(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(h, "Bearer ")
			if !ok || token == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="hefesto"`)
				writeError(w, r, auth.ErrInvalidToken)
				return
			}
			p, err := svc.Authenticate(token)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="hefesto", error="invalid_token"`)
				writeError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
		})
	}
}

// clientIP is the address the request came from. Behind Caddy the peer is
// the proxy, so with trustProxy the rightmost X-Forwarded-For entry — the one
// our own proxy appended — is used instead. Leftmost entries are
// client-supplied and never trusted.
func clientIP(r *http.Request, trustProxy bool) *netip.Addr {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if a, err := netip.ParseAddr(strings.TrimSpace(parts[len(parts)-1])); err == nil {
				return &a
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if a, err := netip.ParseAddr(host); err == nil {
		return &a
	}
	return nil
}

// ipLimiter is a per-address token bucket for the credential endpoints. It is
// in memory, which is right for one API process; several would each allow
// the full rate.
type ipLimiter struct {
	mu         sync.Mutex
	perMinute  float64
	burst      int
	trustProxy bool
	buckets    map[netip.Addr]*bucket
	calls      int
}

type bucket struct {
	lim  *rate.Limiter
	seen time.Time
}

func newIPLimiter(perMinute float64, burst int, trustProxy bool) *ipLimiter {
	return &ipLimiter{perMinute: perMinute, burst: burst, trustProxy: trustProxy, buckets: map[netip.Addr]*bucket{}}
}

func (l *ipLimiter) allow(ip netip.Addr) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.calls++
	if l.calls%1000 == 0 {
		for k, b := range l.buckets {
			if now.Sub(b.seen) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
	}
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{lim: rate.NewLimiter(rate.Limit(l.perMinute/60), l.burst)}
		l.buckets[ip] = b
	}
	b.seen = now
	res := b.lim.ReserveN(now, 1)
	if d := res.DelayFrom(now); d > 0 {
		res.CancelAt(now)
		return false, d
	}
	return true, 0
}

func (l *ipLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, l.trustProxy)
		if ip != nil {
			if ok, wait := l.allow(*ip); !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
				p := problem("rate-limited", "Too many requests", http.StatusTooManyRequests, "")
				p.TraceID = RequestIDFrom(r.Context())
				WriteProblem(w, r, p)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
