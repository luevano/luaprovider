package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luevano/libmangal/netutil"
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
	// the site didn't say, mangal waits as long as the next retry would have
	if rateLimit != nil && (rateLimit.RetryAfter != pageRetryWait<<pageRetries || rateLimit.URL != page("limited.png").URL) {
		t.Errorf("always 429: retry after %s for %s, want %s", rateLimit.RetryAfter, rateLimit.URL, pageRetryWait<<pageRetries)
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

func TestPageHeaders(t *testing.T) {
	received := make(chan http.Header, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		if r.URL.Path == "/broken.png" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("image"))
	}))
	defer server.Close()

	p := loadTestProvider(t, fmt.Sprintf(`
function SearchMangas(query) return {{id = "1", title = "M"}} end
function MangaVolumes(manga) return {{number = 1}} end
function VolumeChapters(volume) return {{title = "One", number = 1, url = "https://site/ch1"}} end
function ChapterPages(chapter)
	return {
		{url = "%[1]s/1.png", headers = {Referer = "custom", ["X-A"] = "a"}, cookies = {sid = "s"}},
		{url = "%[1]s/broken.png"},
	}
end
`, server.URL))
	ctx := context.Background()
	mangas, err := p.SearchMangas(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	volumes, _ := p.MangaVolumes(ctx, mangas[0])
	chapters, _ := p.VolumeChapters(ctx, volumes[0])
	pages, err := p.ChapterPages(ctx, chapters[0])
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages: %v %v", pages, err)
	}

	if _, err := p.GetPageImage(ctx, pages[0]); err != nil {
		t.Fatal(err)
	}
	h := <-received
	for key, want := range map[string]string{
		"Referer":    "custom",
		"X-A":        "a",
		"User-Agent": netutil.BrowserUserAgent,
		"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
		"Cookie":     "sid=s",
	} {
		if got := h.Get(key); got != want {
			t.Errorf("%s: got %q, wanted %q", key, got, want)
		}
	}

	_, err = p.GetPageImage(ctx, pages[1])
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("500: wanted an HTTPError, got %v", err)
	}
	if h := <-received; h.Get("Referer") != "https://site/ch1" {
		t.Errorf("default referer: got %q, wanted the chapter url", h.Get("Referer"))
	}
}
