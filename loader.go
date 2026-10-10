package luaprovider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	lm "github.com/luevano/libmangal"
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
}

func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{Timeout: time.Minute},
		UserAgent:  defaultUserAgent,
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
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
	return p, nil
}

func (l loader) load(ctx context.Context) (_ *provider, err error) {
	options := l.options
	if options.UserAgent == "" {
		options.UserAgent = defaultUserAgent
	}
	log := newProviderLogger(l.info.ID + ": ")

	state, store, err := newState(options, l.info.ID, log)
	if err != nil {
		return nil, err
	}
	// don't leave the state and the cache store open if the script fails
	defer func() {
		if err != nil {
			state.Close()
			store.Close()
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
		return nil, err
	}

	p := &provider{
		info:    l.info,
		options: options,
		log:     log,
		state:   state,
		store:   store,
	}
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
	return p, nil
}
