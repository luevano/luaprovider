package headless

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const elementTypeName = libName + "_element"

func elementInput(state *lua.LState) int {
	element := util.Check[*rod.Element](state, 1)
	text := state.CheckString(2)

	if err := element.Context(state.Context()).Input(text); err != nil {
		util.Raise(state, fmt.Errorf("headless element input: %w", err))
	}

	return 0
}

func elementClick(state *lua.LState) int {
	element := util.Check[*rod.Element](state, 1)
	count := state.OptNumber(2, 1)

	if err := element.Context(state.Context()).Click(proto.InputMouseButtonLeft, int(count)); err != nil {
		util.Raise(state, fmt.Errorf("headless element click: %w", err))
	}

	return 0
}

func elementHTML(state *lua.LState) int {
	element := util.Check[*rod.Element](state, 1)

	HTML, err := element.Context(state.Context()).HTML()
	if err != nil {
		util.Raise(state, fmt.Errorf("headless element html: %w", err))
		return 0
	}

	state.Push(lua.LString(HTML))
	return 1
}
