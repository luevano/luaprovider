package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/luaprovider/lib"
	luahttp "github.com/luevano/luaprovider/lib/http"
	"github.com/luevano/luaprovider/util"
	"github.com/philippgille/gokv"
	"github.com/samber/lo"
	lua "github.com/yuin/gopher-lua"
)

func newState(options Options, limiter *luahttp.Limiter, store gokv.Store, log *logger.ProviderLogger, resources *resources) *lua.LState {
	libs := []lua.LGFunction{
		lua.OpenBase,
		lua.OpenTable,
		lua.OpenString,
		lua.OpenMath,
		lua.OpenPackage,
		lua.OpenCoroutine,
		openChannel,
	}

	state := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})

	for _, injectLib := range libs {
		injectLib(state)
	}

	lib.Preload(state, &lib.Options{
		HTTPClient:   options.HTTPClient,
		RateLimiter:  limiter,
		CacheStore:   store,
		UserAgent:    options.UserAgent,
		Log:          log.Log,
		Status:       func(ctx context.Context, message string) { log.Status(ctx, "%s", message) },
		Track:        resources.track,
		Filter:       options.Filter,
		FlareSolverr: options.FlareSolverr,
		Debug:        options.Debug,
	})

	pkg := state.GetGlobal("package").(*lua.LTable)

	paths := lo.Map(options.PackagePaths, func(path string, _ int) string {
		return filepath.Join(path, "?.lua")
	})

	pkg.RawSetString("path", lua.LString(strings.Join(paths, ";")))

	sandbox(state, options.PackagePaths, log)
	return state
}

// maxChannelBuffer caps channel.make(n), a huge n would allocate it all at once.
const maxChannelBuffer = 1 << 16

// openChannel is gopher-lua's channel lib with a send that stops when the call is cancelled
// (gopher-lua's blocks forever, wedging the provider) and a capped buffer.
func openChannel(L *lua.LState) int {
	n := lua.OpenChannel(L)
	mod := L.Get(-1).(*lua.LTable)
	makeChannel := mod.RawGetString("make").(*lua.LFunction)
	mod.RawSetString("make", L.NewFunction(func(L *lua.LState) int {
		if n := L.OptInt(1, 0); n < 0 || n > maxChannelBuffer {
			L.ArgError(1, fmt.Sprintf("buffer must be between 0 and %d, got %d", maxChannelBuffer, n))
		}
		return makeChannel.GFunction(L)
	}))
	mt := L.GetMetatable(lua.LChannel(nil)).(*lua.LTable)
	mt.RawSetString("send", L.NewFunction(channelSend))
	return n
}

func channelSend(L *lua.LState) int {
	ch := L.CheckChannel(1)
	v := L.CheckAny(2)
	switch v := v.(type) {
	case *lua.LFunction, *lua.LUserData, *lua.LState:
		L.ArgError(2, "can not send a function, userdata, thread or table that has a metatable")
	case *lua.LTable:
		if v.Metatable != lua.LNil {
			L.ArgError(2, "can not send a function, userdata, thread or table that has a metatable")
		}
	}
	ctx := L.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case ch <- v:
	case <-ctx.Done():
		util.Raise(L, ctx.Err())
	}
	return 0
}

// sandbox keeps scripts away from the file system and stdout: no io/os, nothing that loads
// files but require (confined to the package paths), print goes to the provider log.
func sandbox(state *lua.LState, packagePaths []string, log *logger.ProviderLogger) {
	for _, name := range []string{"dofile", "loadfile", "load", "loadstring", "_printregs"} {
		state.SetGlobal(name, lua.LNil)
	}

	state.SetGlobal("print", state.NewFunction(func(L *lua.LState) int {
		parts := make([]string, L.GetTop())
		for i := range parts {
			parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
		}
		log.Log("%s", strings.Join(parts, "\t"))
		return 0
	}))

	// the second loader is gopher-lua's file loader, which follows package.path (scripts
	// can change it) and reads any file
	loaders := state.GetField(state.GetGlobal("package"), "loaders").(*lua.LTable)
	loaders.RawSetInt(2, state.NewFunction(fileLoader(packagePaths)))
}

var moduleName = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$`)

// fileLoader is a package loader for "a.b" as a/b.lua in one of dirs, and nothing outside them.
func fileLoader(dirs []string) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.CheckString(1)
		if !moduleName.MatchString(name) {
			L.Push(lua.LString(fmt.Sprintf("invalid module name %q", name)))
			return 1
		}
		rel := filepath.Join(strings.Split(name, ".")...) + ".lua"

		var messages []string
		for _, dir := range dirs {
			path, err := confinedPath(dir, rel)
			if err != nil {
				messages = append(messages, err.Error())
				continue
			}
			fn, err := L.LoadFile(path)
			if err != nil {
				util.Raise(L, err)
			}
			L.Push(fn)
			return 1
		}
		L.Push(lua.LString(strings.Join(messages, "\n\t")))
		return 1
	}
}

// confinedPath is rel in dir, an error if it doesn't exist or resolves (symlinks) outside dir.
func confinedPath(dir, rel string) (string, error) {
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	if inside, err := filepath.Rel(root, path); err != nil || !filepath.IsLocal(inside) {
		return "", fmt.Errorf("%s is outside %s", rel, dir)
	}
	return path, nil
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
