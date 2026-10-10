package lib

import (
	"io"
	"net/http"
	"time"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/lib/crypto"
	"github.com/luevano/luaprovider/lib/encoding"
	"github.com/luevano/luaprovider/lib/headless"
	"github.com/luevano/luaprovider/lib/html"
	httpLib "github.com/luevano/luaprovider/lib/http"
	"github.com/luevano/luaprovider/lib/js"
	"github.com/luevano/luaprovider/lib/levenshtein"
	"github.com/luevano/luaprovider/lib/log"
	optionsLib "github.com/luevano/luaprovider/lib/options"
	"github.com/luevano/luaprovider/lib/regexp"
	"github.com/luevano/luaprovider/lib/strings"
	timeLib "github.com/luevano/luaprovider/lib/time"
	"github.com/luevano/luaprovider/lib/urls"
	"github.com/luevano/luaprovider/lib/util"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
	lua "github.com/yuin/gopher-lua"
)

type Options struct {
	HTTPClient *http.Client
	CacheStore gokv.Store
	// UserAgent: Default User-Agent header of http requests.
	UserAgent string
	// Log: Logs a message of the provider, used by the log library.
	Log func(format string, a ...any)
	// Track: Resources (e.g. headless browsers) to close with the provider.
	Track func(io.Closer)
	// Filter: The user's filters, read by scripts from the options library.
	Filter optionsLib.Filter
}

func DefaultOptions() *Options {
	return &Options{
		HTTPClient: &http.Client{Timeout: time.Minute},
		CacheStore: syncmap.NewStore(syncmap.DefaultOptions),
		Log:        func(string, ...any) {},
		Track:      func(io.Closer) {},
		Filter:     optionsLib.Filter{Language: "en"},
	}
}

const libName = "sdk"

func Lib(L *lua.LState, options *Options) *luadoc.Lib {
	return &luadoc.Lib{
		Name:        libName,
		Description: `Contains various utilities for making HTTP requests, working with JSON, HTML, and more.`,
		Libs: []*luadoc.Lib{
			regexp.Lib(L),
			strings.Lib(),
			crypto.Lib(L),
			js.Lib(),
			html.Lib(),
			levenshtein.Lib(),
			util.Lib(),
			timeLib.Lib(),
			urls.Lib(),
			encoding.Lib(L),
			headless.Lib(options.Track),
			httpLib.Lib(httpLib.Options{
				HTTPClient: options.HTTPClient,
				CacheStore: options.CacheStore,
				UserAgent:  options.UserAgent,
			}),
			log.Lib(options.Log),
			optionsLib.Lib(options.Filter),
		},
	}
}

func Preload(L *lua.LState, options *Options) {
	lib := Lib(L, options)
	L.PreloadModule(lib.Name, lib.Loader())
}
