package headless

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const pageTypeName = libName + "_page"

func pageElement(state *lua.LState) int {
	page := util.Check[*rod.Page](state, 1)
	selector := state.CheckString(2)

	found, element, err := page.Context(state.Context()).Has(selector)
	if err != nil {
		util.Raise(state, fmt.Errorf("headless element %q: %w", selector, err))
		return 0
	}
	if !found {
		return 0
	}
	util.Push(state, element, elementTypeName)
	return 1
}

func pageNavigate(state *lua.LState) int {
	page := util.Check[*rod.Page](state, 1)
	URL := state.CheckString(2)

	if err := page.Context(state.Context()).Navigate(URL); err != nil {
		util.Raise(state, fmt.Errorf("headless navigate %s: %w", URL, err))
	}
	return 0
}

func pageHTML(state *lua.LState) int {
	page := util.Check[*rod.Page](state, 1)

	HTML, err := page.Context(state.Context()).HTML()
	if err != nil {
		util.Raise(state, fmt.Errorf("headless page html: %w", err))
		return 0
	}
	state.Push(lua.LString(HTML))
	return 1
}

func pageClose(state *lua.LState) int {
	page := util.Check[*rod.Page](state, 1)
	if err := page.Close(); err != nil {
		util.Raise(state, fmt.Errorf("headless page close: %w", err))
	}
	return 0
}
