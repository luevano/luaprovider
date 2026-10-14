package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRateLimit(t *testing.T) {
	for in, want := range map[string]RateLimit{
		"1/2s":    {Permits: 1, Period: 2 * time.Second},
		"5/s":     {Permits: 5, Period: time.Second},
		" 40/1m":  {Permits: 40, Period: time.Minute},
		"3/500ms": {Permits: 3, Period: 500 * time.Millisecond},
	} {
		got, err := ParseRateLimit(in)
		if err != nil || got != want {
			t.Errorf("%q: got %+v, %v, want %+v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "5", "0/s", "-1/s", "1/0s", "1/-2s", "x/s", "1/x"} {
		if got, err := ParseRateLimit(in); err == nil {
			t.Errorf("%q: got %+v, want an error", in, got)
		}
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestRateLimitMatch(t *testing.T) {
	for _, c := range []struct {
		match, url string
		want       bool
	}{
		{"", "https://any.host/x", true},
		{"site.com", "https://site.com/a", true},
		{"site.com", "https://SITE.com:8080/a", true},
		{"site.com", "https://img.site.com/a", false},
		{"site.com", "https://othersite.com/a", false},
		{"api.site.com/at-home/", "https://api.site.com/at-home/server/1", true},
		{"api.site.com/at-home/", "https://api.site.com/manga", false},
		{"api.site.com/at-home", "https://api.site.com/at-home", true},
	} {
		rule, err := newRule(RateLimit{Permits: 1, Period: time.Second, Match: c.match})
		if err != nil {
			t.Fatal(err)
		}
		if got := rule.matches(mustURL(t, c.url)); got != c.want {
			t.Errorf("%q matches %q: got %v", c.match, c.url, got)
		}
	}
}

func TestLimiterAddValidates(t *testing.T) {
	l := NewLimiter(nil)
	for _, r := range []RateLimit{
		{Permits: 0, Period: time.Second},
		{Permits: 1, Period: 0},
		{Permits: 1, Period: time.Second, Burst: -1},
		{Permits: 1, Period: time.Second, Match: "/path"},
	} {
		if err := l.Add(r); err == nil {
			t.Errorf("%+v: want an error", r)
		}
	}
}

func TestLimiterWaits(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer server.Close()

	var statuses []string
	l := NewLimiter(func(_ context.Context, message string) { statuses = append(statuses, message) })
	l.statusAfter = 50 * time.Millisecond
	host := mustURL(t, server.URL).Host
	if err := l.Add(RateLimit{Permits: 1, Period: 100 * time.Millisecond, Match: host}); err != nil {
		t.Fatal(err)
	}
	// another host, never waited for
	if err := l.Add(RateLimit{Permits: 1, Period: time.Hour, Match: "other.host"}); err != nil {
		t.Fatal(err)
	}
	client := l.Client(server.Client())

	start := time.Now()
	for range 3 {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Errorf("3 requests at 1 per 100ms took %s", elapsed)
	}
	if hits.Load() != 3 {
		t.Errorf("hits %d", hits.Load())
	}
	if len(statuses) != 2 || !strings.Contains(statuses[0], "rate limit") || !strings.Contains(statuses[0], host) {
		t.Errorf("statuses %q, want 2 waits on %s", statuses, host)
	}
}

func TestLimiterWaitStopsOnCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	l := NewLimiter(nil)
	if err := l.Add(RateLimit{Permits: 1, Period: time.Hour}); err != nil {
		t.Fatal(err)
	}
	client := l.Client(server.Client())
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	start := time.Now()
	if _, err := client.Do(req); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled while waiting: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("waited %s after the cancel", elapsed)
	}
}

func TestLimiterClientKeepsTheOriginal(t *testing.T) {
	base := &http.Client{Timeout: time.Minute}
	client := NewLimiter(nil).Client(base)
	if client == base || base.Transport != nil || client.Timeout != time.Minute {
		t.Errorf("client %+v, base %+v", client, base)
	}
}
