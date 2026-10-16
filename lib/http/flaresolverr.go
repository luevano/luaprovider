package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/luevano/libmangal/netutil"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

// ErrNoFlareSolverr is raised by request:flaresolverr() when the app didn't configure FlareSolverr.
var ErrNoFlareSolverr = errors.New("flaresolverr isn't configured")

type flareSolverrKey struct{}

// flareSolverr makes the transport (one FlareSolverr session) on the first request that asks
// for it; it's closed with the provider.
type flareSolverr struct {
	newTransport func() http.RoundTripper
	limiter      *Limiter
	track        func(io.Closer)
	debug        func(format string, a ...any)

	mu     sync.Mutex
	client *http.Client
}

func (f *flareSolverr) available() bool {
	return f != nil && f.newTransport != nil
}

func (f *flareSolverr) get() *http.Client {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.client != nil {
		return f.client
	}
	transport := f.newTransport()
	if closer, ok := transport.(io.Closer); ok && f.track != nil {
		f.track(closer)
	}
	if f.debug != nil {
		debug := f.debug
		transport = netutil.NewLoggingTransport(transport, func(format string, a ...any) {
			debug("flaresolverr "+format, a...)
		}, nil)
	}
	// no timeout: each solve has its own
	client := &http.Client{Transport: transport}
	if f.limiter != nil {
		client = f.limiter.Client(client)
	}
	f.client = client
	return client
}

func requestFlareSolverr(f *flareSolverr) lua.LGFunction {
	return func(L *lua.LState) int {
		request := util.Check[*http.Request](L, 1)
		if !f.available() {
			util.Raise(L, ErrNoFlareSolverr)
			return 0
		}
		*request = *request.WithContext(context.WithValue(request.Context(), flareSolverrKey{}, true))
		L.Push(L.Get(1))
		return 1
	}
}

func viaFlareSolverr(request *http.Request) bool {
	via, _ := request.Context().Value(flareSolverrKey{}).(bool)
	return via
}

func hasFlareSolverr(f *flareSolverr) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(lua.LBool(f.available()))
		return 1
	}
}
