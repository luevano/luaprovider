package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCloseStopsTheRunningCall(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-r.Context().Done()
	}))
	defer server.Close()

	p := loadTestProvider(t, fmt.Sprintf(`
local sdk = require("sdk")
function SearchMangas(query)
	sdk.http.request("GET", %q):send()
	return {}
end
`, server.URL)+"function MangaVolumes"+strings.SplitN(testScript, "function MangaVolumes", 2)[1])

	errs := make(chan error, 1)
	go func() {
		_, err := p.SearchMangas(context.Background(), "x")
		errs <- err
	}()
	<-started

	closed := make(chan error, 1)
	go func() { closed <- p.Close() }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close waited for the running call")
	}
	select {
	case err := <-errs:
		if err == nil {
			t.Error("the cancelled call should fail")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close didn't cancel the running call")
	}

	if _, err := p.SearchMangas(context.Background(), "x"); !errors.Is(err, ErrClosed) {
		t.Errorf("after Close: %v, want ErrClosed", err)
	}
}

func TestCancelledCallReturnsTheContextError(t *testing.T) {
	p := loadTestProvider(t, `
function SearchMangas(query)
	while true do end
end
`+"function MangaVolumes"+strings.SplitN(testScript, "function MangaVolumes", 2)[1])

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := p.SearchMangas(ctx, "x"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("timed out: %v, want context.DeadlineExceeded", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	if _, err := p.SearchMangas(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, want context.Canceled", err)
	}
}
