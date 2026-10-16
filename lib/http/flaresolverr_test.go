package http_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luevano/luaprovider/lib"
	luahttp "github.com/luevano/luaprovider/lib/http"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

// fakeFlareSolverr answers every request itself, like a FlareSolverr session would.
type fakeFlareSolverr struct {
	requests atomic.Int32
	closed   atomic.Int32
}

func (f *fakeFlareSolverr) RoundTrip(r *http.Request) (*http.Response, error) {
	f.requests.Add(1)
	body := "solved " + r.Method + " " + r.URL.Path
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/html"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func (f *fakeFlareSolverr) Close() error {
	f.closed.Add(1)
	return nil
}

func TestFlareSolverr(t *testing.T) {
	var direct atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		direct.Add(1)
		fmt.Fprint(w, "direct")
	}))
	defer server.Close()

	fake := &fakeFlareSolverr{}
	var made atomic.Int32
	var tracked []io.Closer
	var debug []string

	L := lua.NewState()
	defer L.Close()
	options := lib.DefaultOptions()
	options.FlareSolverr = func() http.RoundTripper {
		made.Add(1)
		return fake
	}
	options.Track = func(c io.Closer) { tracked = append(tracked, c) }
	options.Debug = func(format string, a ...any) { debug = append(debug, fmt.Sprintf(format, a...)) }
	options.RateLimiter = luahttp.NewLimiter(nil)
	if err := options.RateLimiter.Add(luahttp.RateLimit{Permits: 1, Period: time.Second, Match: "challenged.test"}); err != nil {
		t.Fatal(err)
	}
	lib.Preload(L, options)
	L.SetContext(context.Background())

	start := time.Now()
	err := L.DoString(fmt.Sprintf(`
local http = require("sdk").http
assert(http.has_flaresolverr(), "has_flaresolverr")
assert(http.request("GET", %q):send():body() == "direct", "direct request")
local r = http.request("GET", "https://challenged.test/manga")
assert(r:flaresolverr() == r, "returns the request")
assert(r:send():body() == "solved GET /manga", "through flaresolverr")
assert(http.request("POST", "https://challenged.test/search", "q=x"):flaresolverr():send():body() == "solved POST /search", "post")
-- cached like any other GET
assert(http.request("GET", "https://challenged.test/manga"):flaresolverr():send():body() == "solved GET /manga", "cached")
`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if direct.Load() != 1 || fake.requests.Load() != 2 || made.Load() != 1 {
		t.Errorf("wanted 1 direct request, 2 through one flaresolverr transport; got %d, %d, %d transports", direct.Load(), fake.requests.Load(), made.Load())
	}
	if len(tracked) != 1 || tracked[0] != io.Closer(fake) {
		t.Errorf("the transport should be tracked to close with the provider, got %v", tracked)
	}
	// the second request to the limited host waits a second
	if time.Since(start) < 900*time.Millisecond {
		t.Error("flaresolverr requests should wait for the rate limits")
	}
	want := []string{"flaresolverr GET https://challenged.test/manga 200", "flaresolverr POST https://challenged.test/search 200", "GET https://challenged.test/manga cached"}
	if len(debug) != len(want) {
		t.Fatalf("debug lines: got %q", debug)
	}
	for i, line := range debug {
		if !strings.HasPrefix(line, want[i]) {
			t.Errorf("debug line %d: got %q, wanted %q...", i, line, want[i])
		}
	}
}

func TestFlareSolverrNotConfigured(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	if err := L.DoString(`assert(not require("sdk").http.has_flaresolverr())`); err != nil {
		t.Fatal(err)
	}
	err := util.GoError(L.DoString(`require("sdk").http.request("GET", "https://site.test"):flaresolverr()`))
	if !errors.Is(err, luahttp.ErrNoFlareSolverr) {
		t.Fatalf("wanted ErrNoFlareSolverr, got %v", err)
	}
}
