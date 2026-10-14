package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luevano/libmangal"
	luahttp "github.com/luevano/luaprovider/lib/http"
)

// rateLimitScript searches with a request to the url in the query, after rate_limit(limit).
func rateLimitScript(limit string) string {
	return fmt.Sprintf(`
local http = require("sdk").http
%s
function SearchMangas(query)
	http.request("POST", query):send()
	return {}
end
function MangaVolumes(manga) return {} end
function VolumeChapters(volume) return {} end
function ChapterPages(chapter) return {} end
`, limit)
}

func TestScriptRateLimit(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("image"))
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")

	// 1 per 100ms on the server's host, shared by the script's requests and the page images
	p := loadTestProvider(t, rateLimitScript(fmt.Sprintf(`http.rate_limit(1, 0.1, %q)`, host)))
	ctx := context.Background()
	start := time.Now()
	if _, err := p.SearchMangas(ctx, server.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := p.SearchMangas(ctx, server.URL); err != nil {
		t.Fatal(err)
	}
	page := &luaPage{URL: server.URL + "/1.png", chapter: &luaChapter{Title: "One"}}
	if _, err := p.GetPageImage(ctx, page); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Errorf("2 searches and a page at 1 per 100ms took %s", elapsed)
	}
	if hits.Load() != 3 {
		t.Errorf("hits %d", hits.Load())
	}

	// another host isn't limited
	other := loadTestProvider(t, rateLimitScript(`http.rate_limit(1, 3600, "other.host")`))
	start = time.Now()
	for range 3 {
		if _, err := other.SearchMangas(ctx, server.URL); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("unlimited host took %s", elapsed)
	}
}

func TestScriptRateLimitStopsOnCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	p := loadTestProvider(t, rateLimitScript(`http.rate_limit(1, 3600)`))
	if _, err := p.SearchMangas(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	if _, err := p.SearchMangas(ctx, server.URL); !errors.Is(err, context.Canceled) {
		t.Errorf("search: %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	page := &luaPage{URL: server.URL + "/1.png", chapter: &luaChapter{Title: "One"}}
	if _, err := p.GetPageImage(ctx, page); !errors.Is(err, context.Canceled) {
		t.Errorf("page: %v", err)
	}
}

func TestScriptRateLimitInvalid(t *testing.T) {
	for _, limit := range []string{
		`http.rate_limit(0, 1)`,
		`http.rate_limit(1, 0)`,
		`http.rate_limit(1, 1, "/path")`,
		`http.rate_limit(1, 1, nil, 0)`,
	} {
		info := testInfo()
		loader, err := NewLoader([]byte(rateLimitScript(limit)), info, DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		if p, err := loader.Load(context.Background()); err == nil {
			p.Close()
			t.Errorf("%s: loaded", limit)
		}
	}
}

func TestOptionsRateLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	options := DefaultOptions()
	options.RateLimits = []luahttp.RateLimit{{Permits: 1, Period: 100 * time.Millisecond, Match: u.Host}}
	p := loadTestProviderWith(t, rateLimitScript(""), options)
	start := time.Now()
	for range 3 {
		if _, err := p.SearchMangas(context.Background(), server.URL); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Errorf("3 searches at 1 per 100ms took %s", elapsed)
	}

	options.RateLimits = []luahttp.RateLimit{{Permits: 0, Period: time.Second}}
	loader, _ := NewLoader([]byte(rateLimitScript("")), testInfo(), options)
	if p, err := loader.Load(context.Background()); err == nil {
		p.Close()
		t.Error("invalid rate limit option: loaded")
	}
}

func testInfo() libmangal.ProviderInfo {
	return libmangal.ProviderInfo{ID: "test", Name: "Test", Version: "0.1.0"}
}
