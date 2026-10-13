package luaprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetPageImageRetriesRateLimits(t *testing.T) {
	defer func(w time.Duration) { pageRetryWait = w }(pageRetryWait)
	pageRetryWait = time.Millisecond

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		switch r.URL.Path {
		case "/missing.png":
			http.NotFound(w, r)
		case "/limited.png":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			if n <= 2 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.Write([]byte("image"))
		}
	}))
	defer server.Close()

	p := loadTestProvider(t, testScript)
	page := func(name string) *luaPage {
		return &luaPage{URL: server.URL + "/" + name, chapter: &luaChapter{Title: "One"}}
	}

	image, err := p.GetPageImage(context.Background(), page("1.png"))
	if err != nil || string(image) != "image" || calls.Load() != 3 {
		t.Errorf("two 429s then the image: %q, %v after %d calls", image, err, calls.Load())
	}

	calls.Store(0)
	_, err = p.GetPageImage(context.Background(), page("limited.png"))
	var rateLimit *RateLimitError
	if !errors.As(err, &rateLimit) || !errors.Is(err, ErrRateLimited) || calls.Load() != pageRetries+1 {
		t.Errorf("always 429: %v after %d calls, want a RateLimitError after %d", err, calls.Load(), pageRetries+1)
	}

	calls.Store(0)
	_, err = p.GetPageImage(context.Background(), page("missing.png"))
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || !errors.Is(err, ErrNotFound) || calls.Load() != 1 {
		t.Errorf("404: %v after %d calls, want a not found HTTPError, no retries", err, calls.Load())
	}

	pageRetryWait = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	if _, err := p.GetPageImage(ctx, page("limited.png")); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled while waiting: %v", err)
	}
}
