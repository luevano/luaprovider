package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luevano/libmangal"
)

func TestHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing":
			http.NotFound(w, r)
		case "/limited":
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}))
	defer server.Close()

	p := loadTestProvider(t, fmt.Sprintf(`
local sdk = require("sdk")

function SearchMangas(query)
	if query == "caught" then
		-- scripts can still catch it and read the message
		local ok, err = pcall(function() sdk.http.request("GET", %[1]q .. "/missing"):send():check() end)
		return {{id = "1", title = tostring(err), url = "x"}}
	end
	-- send raises for 429, check for the rest
	sdk.http.request("GET", %[1]q .. "/" .. query):send():check()
	return {}
end
`, server.URL)+"function MangaVolumes"+strings.SplitN(testScript, "function MangaVolumes", 2)[1])

	_, err := p.SearchMangas(context.Background(), "missing")
	var httpErr *HTTPError
	if !errors.Is(err, ErrNotFound) || !errors.Is(err, libmangal.ErrNotFound) || !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		t.Errorf("missing: %v, want a not found HTTPError", err)
	}

	// only a 429 looks like a rate limit to apps that check for RetryAfterDuration
	var retrier interface{ RetryAfterDuration() time.Duration }
	if errors.As(err, &retrier) {
		t.Errorf("missing: %v shouldn't have a RetryAfterDuration", err)
	}

	_, err = p.SearchMangas(context.Background(), "limited")
	if !errors.Is(err, ErrRateLimited) || !errors.Is(err, libmangal.ErrRateLimited) || !errors.As(err, &retrier) || retrier.RetryAfterDuration() != 3*time.Second {
		t.Errorf("limited: %v, want rate limited for 3s", err)
	}
	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) || errors.Is(err, ErrNotFound) {
		t.Errorf("limited: %v, want a RateLimitError", err)
	}

	mangas, err := p.SearchMangas(context.Background(), "caught")
	if err != nil || len(mangas) != 1 || !strings.Contains(mangas[0].Info().Title, "404 not found") {
		t.Errorf("caught: %v, %v", mangas, err)
	}
}
