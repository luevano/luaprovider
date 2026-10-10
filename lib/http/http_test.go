package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

func TestSendCacheAndBody(t *testing.T) {
	var hits atomic.Int32
	var userAgent atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		userAgent.Store(r.Header.Get("User-Agent"))
		fmt.Fprint(w, "hello")
	}))
	defer server.Close()

	L := lua.NewState()
	defer L.Close()
	options := lib.DefaultOptions()
	options.UserAgent = "test-agent"
	lib.Preload(L, options)
	L.SetContext(context.Background())

	script := fmt.Sprintf(`
local sdk = require("sdk")
local function get(method)
	local response = sdk.http.request(method, %q):send()
	-- the body can be read twice
	assert(response:body() == "hello" and response:body() == "hello", "body")
end
get("GET")
get("GET")
get("POST")
get("POST")
`, server.URL)
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}

	// the second GET comes from the cache, POSTs aren't cached
	if got := hits.Load(); got != 3 {
		t.Errorf("wanted 3 requests to the server, got %d", got)
	}
	if got := userAgent.Load(); got != "test-agent" {
		t.Errorf("wanted the default user agent, got %v", got)
	}
}
