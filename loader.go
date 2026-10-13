package luaprovider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	lm "github.com/luevano/libmangal"
	"github.com/luevano/luaprovider/lib/options"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
	lua "github.com/yuin/gopher-lua"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0"

var _ lm.ProviderLoader = (*loader)(nil)

// Options are the general luaprovider options.
type Options struct {
	// HTTPClient HTTP client to use for all requests.
	HTTPClient *http.Client

	// UserAgent to use for all HTTP requests.
	UserAgent string

	// CacheStore returns a gokv.Store implementation for use as a cache storage.
	CacheStore func(dbName, bucketName string) (gokv.Store, error)

	// PackagePaths is a list of optional extra package paths.
	PackagePaths []string

	// Filter options, scripts read them from sdk.options.
	Filter FilterOptions
}

// FilterOptions are the same as mangoprovider's.
type FilterOptions = options.Filter

func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		NSFW:                    false,
		Language:                "en",
		AvoidDuplicateChapters:  true,
		ShowUnavailableChapters: false,
	}
}

func DefaultOptions() Options {
	// shared by every provider loaded with these options
	store := syncmap.NewStore(syncmap.DefaultOptions)
	return Options{
		HTTPClient: &http.Client{Timeout: time.Minute},
		UserAgent:  defaultUserAgent,
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
	log := newProviderLogger(l.info)
	res := &resources{}

	// opened once, closed with the provider
	store, err := options.CacheStore(l.info.ID, l.info.ID)
	if err != nil {
		return nil, fmt.Errorf("open cache: %w", err)
	}
	res.track(store)

	state := newState(options, store, log, res)
	// don't leave the state open if the script fails
	defer func() {
		if err != nil {
			res.close()
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
