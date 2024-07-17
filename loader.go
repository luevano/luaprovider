package luaprovider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	lm "github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
	lua "github.com/yuin/gopher-lua"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0"

var _ lm.ProviderLoader = (*loader)(nil)

// TODO: use the cache store effectively like with mangoprovider

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
		HTTPClient: &http.Client{},
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
	provider := &provider{
		info:    l.info,
		options: l.options,
	}

	state, store, err := newState(l.options, l.info.ID)
	if err != nil {
		return nil, err
	}

	provider.store = store
	provider.state = state
	provider.state.SetContext(ctx)
	lfunc, err := provider.state.Load(bytes.NewReader(l.script), l.info.Name)
	if err != nil {
		return nil, err
	}

	err = provider.state.CallByParam(lua.P{
		Fn:      lfunc,
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		return nil, err
	}

	for name, fn := range map[string]**lua.LFunction{
		methodSearchMangas:   &provider.fnSearchMangas,
		methodMangaVolumes:   &provider.fnMangaVolumes,
		methodVolumeChapters: &provider.fnVolumeChapters,
		methodChapterPages:   &provider.fnChapterPages,
	} {
		var found bool
		*fn, found = provider.state.GetGlobal(name).(*lua.LFunction)

		if !found {
			return nil, fmt.Errorf("missing function: %s", name)
		}
	}

	return provider, nil
}
