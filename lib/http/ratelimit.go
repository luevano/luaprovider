package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/time/rate"
)

// RateLimit is a limit on a provider's requests: Permits per Period, like keiyoushi's
// rateLimit(permits, period) (and rateLimitHost with Match).
type RateLimit struct {
	Permits int
	Period  time.Duration
	// Burst is how many requests can go at once, 1 if zero: a bigger burst lets more than
	// Permits through in a Period.
	Burst int
	// Match is the host the limit applies to, optionally followed by a path prefix
	// ("api.site.com/at-home/"), every request if empty. The host has to be the same,
	// subdomains aren't included.
	Match string
}

// ParseRateLimit parses "permits/period" (period is a Go duration, a unit alone is one of
// it): "1/2s", "5/s", "40/1m".
func ParseRateLimit(s string) (RateLimit, error) {
	permits, period, ok := strings.Cut(strings.TrimSpace(s), "/")
	if !ok {
		return RateLimit{}, fmt.Errorf("rate limit %q: want permits/period, e.g. 1/2s", s)
	}
	n, err := strconv.Atoi(permits)
	if err != nil {
		return RateLimit{}, fmt.Errorf("rate limit %q: permits: %w", s, err)
	}
	if period != "" && (period[0] < '0' || period[0] > '9') {
		period = "1" + period
	}
	d, err := time.ParseDuration(period)
	if err != nil {
		return RateLimit{}, fmt.Errorf("rate limit %q: period: %w", s, err)
	}
	r := RateLimit{Permits: n, Period: d}
	if err := r.validate(); err != nil {
		return RateLimit{}, fmt.Errorf("rate limit %q: %w", s, err)
	}
	return r, nil
}

func (r RateLimit) validate() error {
	switch {
	case r.Permits < 1:
		return errors.New("permits must be at least 1")
	case r.Period <= 0:
		return errors.New("period must be positive")
	case r.Burst < 0:
		return errors.New("burst can't be negative")
	case strings.HasPrefix(r.Match, "/"):
		return errors.New("match needs a host before the path")
	}
	return nil
}

func (r RateLimit) String() string {
	s := fmt.Sprintf("%d/%s", r.Permits, r.Period)
	if r.Match != "" {
		s += " on " + r.Match
	}
	return s
}

type rule struct {
	RateLimit
	host, path string
	limiter    *rate.Limiter
}

func newRule(r RateLimit) (*rule, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	burst := max(r.Burst, 1)
	host, path, hasPath := strings.Cut(r.Match, "/")
	if hasPath {
		path = "/" + path
	}
	return &rule{
		RateLimit: r,
		host:      strings.ToLower(host),
		path:      path,
		limiter:   rate.NewLimiter(rate.Every(r.Period/time.Duration(r.Permits)), burst),
	}, nil
}

func (r *rule) matches(u *url.URL) bool {
	if r.host != "" && !strings.EqualFold(u.Hostname(), r.host) && !strings.EqualFold(u.Host, r.host) {
		return false
	}
	return strings.HasPrefix(u.Path, r.path)
}

// Limiter holds a provider's rate limits, its Client waits for them before each request.
// Safe for concurrent use, limits can be added any time.
type Limiter struct {
	mu    sync.Mutex
	rules []*rule
	// status reports the waits of at least statusAfter
	status      func(ctx context.Context, message string)
	statusAfter time.Duration
}

// NewLimiter returns a limiter without limits. status, if not nil, gets the waits of a second
// or more, for the provider's status.
func NewLimiter(status func(ctx context.Context, message string)) *Limiter {
	if status == nil {
		status = func(context.Context, string) {}
	}
	return &Limiter{status: status, statusAfter: time.Second}
}

// Add adds a limit, requests wait for every limit they match.
func (l *Limiter) Add(limit RateLimit) error {
	r, err := newRule(limit)
	if err != nil {
		return fmt.Errorf("rate limit %s: %w", limit, err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rules = append(l.rules, r)
	return nil
}

// Wait waits until a request to u is allowed by every limit it matches, or ctx is done.
func (l *Limiter) Wait(ctx context.Context, u *url.URL) error {
	l.mu.Lock()
	var rules []*rule
	for _, r := range l.rules {
		if r.matches(u) {
			rules = append(rules, r)
		}
	}
	l.mu.Unlock()
	if len(rules) == 0 {
		return nil
	}

	now := time.Now()
	var (
		delay        time.Duration
		slowest      *rule
		reservations = make([]*rate.Reservation, len(rules))
	)
	for i, r := range rules {
		// burst is at least 1, so a single reservation is always ok
		reservations[i] = r.limiter.ReserveN(now, 1)
		if d := reservations[i].DelayFrom(now); d > delay {
			delay, slowest = d, r
		}
	}
	if delay == 0 {
		return nil
	}
	if delay >= l.statusAfter {
		l.status(ctx, fmt.Sprintf("rate limit %s, waiting %s", slowest, delay.Round(100*time.Millisecond)))
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		for _, r := range reservations {
			r.Cancel()
		}
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Client returns a copy of client whose requests wait for the limits first.
func (l *Limiter) Client(client *http.Client) *http.Client {
	c := *client
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.Transport = &transport{limiter: l, base: base}
	return &c
}

type transport struct {
	limiter *Limiter
	base    http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context(), req.URL); err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	return t.base.RoundTrip(req)
}

func rateLimit(limiter *Limiter) lua.LGFunction {
	return func(L *lua.LState) int {
		limit := RateLimit{
			Permits: L.CheckInt(1),
			Period:  time.Duration(float64(L.CheckNumber(2)) * float64(time.Second)),
			Match:   L.OptString(3, ""),
			Burst:   L.OptInt(4, 1),
		}
		if limit.Burst < 1 {
			util.Raise(L, fmt.Errorf("rate limit %s: burst must be at least 1", limit))
		}
		if err := limiter.Add(limit); err != nil {
			util.Raise(L, err)
		}
		return 0
	}
}
