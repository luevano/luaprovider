package luaprovider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestChannelSendStopsOnCancel(t *testing.T) {
	p := loadTestProvider(t, `
function SearchMangas(query)
	if query == "big" then channel.make(1e9) end
	local ch = channel.make(1)
	ch:send(1)
	assert(ch:receive())
	ch:send(1)
	ch:send(2)
	return {}
end
`+"function MangaVolumes"+strings.SplitN(testScript, "function MangaVolumes", 2)[1])

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := p.SearchMangas(ctx, "x")
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("blocked send: %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a blocked send ignored the cancelled context")
	}

	if _, err := p.SearchMangas(context.Background(), "big"); err == nil || !strings.Contains(err.Error(), "buffer must be") {
		t.Errorf("huge buffer: %v", err)
	}
}
