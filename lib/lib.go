package lib

import (
	"net/http"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/lib/crypto"
	"github.com/luevano/luaprovider/lib/encoding"
	"github.com/luevano/luaprovider/lib/headless"
	"github.com/luevano/luaprovider/lib/html"
	httpLib "github.com/luevano/luaprovider/lib/http"
	"github.com/luevano/luaprovider/lib/js"
	"github.com/luevano/luaprovider/lib/levenshtein"
	"github.com/luevano/luaprovider/lib/regexp"
	"github.com/luevano/luaprovider/lib/strings"
	"github.com/luevano/luaprovider/lib/time"
	"github.com/luevano/luaprovider/lib/urls"
	"github.com/luevano/luaprovider/lib/util"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
	lua "github.com/yuin/gopher-lua"
)

type Options struct {
	HTTPClient *http.Client
	CacheStore gokv.Store
}

func DefaultOptions() *Options {
	return &Options{
		HTTPClient: &http.Client{},
		CacheStore: syncmap.NewStore(syncmap.DefaultOptions),
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
			time.Lib(),
			urls.Lib(),
			encoding.Lib(L),
			headless.Lib(),
			httpLib.Lib(httpLib.Options{
				HTTPClient: options.HTTPClient,
				CacheStore: options.CacheStore,
			}),
		},
	}
}

func Preload(L *lua.LState, options *Options) {
	lib := Lib(L, options)
	L.PreloadModule(lib.Name, lib.Loader())
}
