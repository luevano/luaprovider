package luaprovider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	lm "github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/netutil"
	luahttp "github.com/luevano/luaprovider/lib/http"
	"github.com/luevano/luaprovider/lib/options"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
	lua "github.com/yuin/gopher-lua"
)

var _ lm.ProviderLoader = (*loader)(nil)

// Options are the general luaprovider options.
type Options struct {
	// HTTPClient HTTP client to use for all requests. It's used as is: keeping a script's
	// requests off private addresses is the app's job (mangal guards it). Headless browsers
	// don't go through it.
	HTTPClient *http.Client

	// UserAgent to use for all HTTP requests.
	UserAgent string

	// CacheStore returns a gokv.Store implementation for use as a cache storage.
	CacheStore func(dbName, bucketName string) (gokv.Store, error)

	// PackagePaths is a list of optional extra package paths.
	PackagePaths []string

	// Filter options, scripts read them from sdk.options.
	Filter FilterOptions

	// RateLimits of the provider's requests (e.g. from its manifest), on top of the ones the
	// script sets with sdk.http.rate_limit. Page images are limited too.
	RateLimits []luahttp.RateLimit

	// FlareSolverr makes the transport scripts' request:flaresolverr() goes through, e.g.
	// mangoprovider's flaresolverr.NewTransportClient(url, client). Called once per provider
	// load, on the first such request; a transport that is an io.Closer (its session) is
	// closed with the provider. Nil: FlareSolverr isn't configured, those requests raise.
	FlareSolverr func() http.RoundTripper

	// Debug logs what doesn't go through HTTPClient: cache hits, FlareSolverr requests and
	// headless page loads, in netutil.LoggingTransport's format. Nothing if nil.
	Debug func(format string, a ...any)
}

// FilterOptions are the same as mangoprovider's.
type FilterOptions = options.Filter

func DefaultFilterOptions() FilterOptions {
	return options.DefaultFilter()
}

func DefaultOptions() Options {
	// shared by every provider loaded with these options
	store := syncmap.NewStore(syncmap.DefaultOptions)
	return Options{
		HTTPClient: netutil.NewClient(),
		UserAgent:  netutil.BrowserUserAgent,
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return store, nil
		},
		Filter: DefaultFilterOptions(),
	}
}

// NewLoader creates new lua provider loader for the given script.
//
// It won't run the script itself.
func NewLoader(script []byte, info lm.ProviderInfo, options Options) (lm.ProviderLoader, error) {
	if err := info.Validate(); err != nil {
		return nil, err
	}

	return loader{
		options: options,
		info:    info,
		script:  script,
	}, nil
}

type loader struct {
	options Options
	info    lm.ProviderInfo
	script  []byte
}

func (l loader) Info() lm.ProviderInfo {
	return l.info
}

func (l loader) String() string {
	return l.info.Name
}

func (l loader) Load(ctx context.Context) (lm.Provider, error) {
	p, err := l.load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", l.info.ID, err)
	}
	// only scripts with MangaByID implement lm.MangaByIDProvider
	if p.fnMangaByID != nil {
		return mangaByIDProvider{p}, nil
	}
	return p, nil
}

func (l loader) load(ctx context.Context) (_ *provider, err error) {
	options := l.options
	defaults := DefaultOptions()
	if options.UserAgent == "" {
		options.UserAgent = defaults.UserAgent
	}
	if options.HTTPClient == nil {
		options.HTTPClient = defaults.HTTPClient
	}
	if options.CacheStore == nil {
		options.CacheStore = defaults.CacheStore
	}
	log := logger.NewProviderLogger(l.info.ID, l.info.Name)
	limiter := luahttp.NewLimiter(func(ctx context.Context, message string) { log.Status(ctx, "%s", message) })
	for _, limit := range options.RateLimits {
		if err := limiter.Add(limit); err != nil {
			return nil, err
		}
	}
	res := &resources{}

	// opened once, closed with the provider
	store, err := options.CacheStore(l.info.ID, l.info.ID)
	if err != nil {
		return nil, fmt.Errorf("open cache: %w", err)
	}
	res.track(store)

	state := newState(options, limiter, store, log, res)
	// don't leave the state open if the script fails
	defer func() {
		if err != nil {
			if closeErr := res.close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close: %w", closeErr))
			}
			state.Close()
		}
	}()

	state.SetContext(ctx)
	lfunc, err := state.Load(bytes.NewReader(l.script), l.info.Name)
	if err != nil {
		return nil, err
	}
	err = state.CallByParam(lua.P{
		Fn:      lfunc,
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		return nil, callError(ctx, err)
	}

	// page images wait for the same limits as the script's requests
	options.HTTPClient = limiter.Client(options.HTTPClient)
	p := &provider{
		info:      l.info,
		options:   options,
		log:       log,
		resources: res,
		state:     state,
	}
	p.closing, p.stopCalls = context.WithCancel(context.Background())
	for name, fn := range map[string]**lua.LFunction{
		methodSearchMangas:   &p.fnSearchMangas,
		methodMangaVolumes:   &p.fnMangaVolumes,
		methodVolumeChapters: &p.fnVolumeChapters,
		methodChapterPages:   &p.fnChapterPages,
	} {
		var found bool
		if *fn, found = state.GetGlobal(name).(*lua.LFunction); !found {
			return nil, fmt.Errorf("missing function %s", name)
		}
	}
	switch fn := state.GetGlobal(methodMangaByID).(type) {
	case *lua.LFunction:
		p.fnMangaByID = fn
	case *lua.LNilType:
	default:
		return nil, fmt.Errorf("%s is a %s, not a function", methodMangaByID, fn.Type())
	}
	return p, nil
}
