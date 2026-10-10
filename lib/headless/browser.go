package headless

import (
	"io"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const browserTypeName = libName + "_browser"

// newNewBrowser: Browsers are tracked so they're closed with the provider.
func newNewBrowser(track func(io.Closer)) lua.LGFunction {
	return func(state *lua.LState) int {
		browser := rod.New()
		if err := browser.Connect(); err != nil {
			state.RaiseError("headless browser: %s", err)
			return 0
		}
		track(browser)

		util.Push(state, browser, browserTypeName)
		return 1
	}
}

func browserPage(state *lua.LState) int {
	browser := util.Check[*rod.Browser](state, 1)
	URL := state.CheckString(2)

	// the browser can outlive a call, each operation uses the context of the current one
	page, err := browser.Context(state.Context()).Page(proto.TargetCreateTarget{URL: URL})
	if err != nil {
		state.RaiseError("headless page %s: %s", URL, err)
		return 0
	}
	if err := page.WaitLoad(); err != nil {
		page.Close()
		state.RaiseError("headless page %s: %s", URL, err)
		return 0
	}

	util.Push(state, page, pageTypeName)
	return 1
}

func browserClose(state *lua.LState) int {
	browser := util.Check[*rod.Browser](state, 1)
	if err := browser.Close(); err != nil {
		state.RaiseError("headless browser close: %s", err)
	}
	return 0
}
