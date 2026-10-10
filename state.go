package luaprovider

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/luevano/luaprovider/lib"
	"github.com/philippgille/gokv"
	"github.com/samber/lo"
	lua "github.com/yuin/gopher-lua"
)

func newState(options Options, providerID string, log *providerLogger, resources *resources) *lua.LState {
	libs := []lua.LGFunction{
		lua.OpenBase,
		lua.OpenTable,
		lua.OpenString,
		lua.OpenMath,
		lua.OpenPackage,
		lua.OpenIo,
		lua.OpenCoroutine,
		lua.OpenChannel,
	}

	state := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})

	for _, injectLib := range libs {
		injectLib(state)
	}

	store := &lazyStore{open: func() (gokv.Store, error) {
		return options.CacheStore(providerID, providerID)
	}}

	lib.Preload(state, &lib.Options{
		HTTPClient: options.HTTPClient,
		CacheStore: store,
		UserAgent:  options.UserAgent,
		Log:        log.log,
		Track:      resources.track,
		Filter:     options.Filter,
	})

	pkg := state.GetGlobal("package").(*lua.LTable)

	paths := lo.Map(options.PackagePaths, func(path string, _ int) string {
		return filepath.Join(path, "?.lua")
	})

	pkg.RawSetString("path", lua.LString(strings.Join(paths, ";")))

	return state
}

// resources: Things the lua libraries open (e.g. headless browsers), closed with the provider.
type resources struct {
	mu      sync.Mutex
	closers []io.Closer
}

func (r *resources) track(closer io.Closer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closers = append(r.closers, closer)
}

func (r *resources) close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var errs []error
	for _, closer := range r.closers {
		errs = append(errs, closer.Close())
	}
	r.closers = nil
	return errors.Join(errs...)
}
