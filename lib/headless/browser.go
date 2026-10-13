package headless

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const browserTypeName = libName + "_browser"

// newNewBrowser: Browsers are tracked so they're closed with the provider.
func newNewBrowser(track func(io.Closer), log func(string, ...any), status func(context.Context, string)) lua.LGFunction {
	return func(state *lua.LState) int {
		u, err := launch(state.Context(), log, status)
		if err != nil {
			state.RaiseError("headless browser: %s", err)
			return 0
		}
		browser := rod.New().ControlURL(u)
		if err := browser.Connect(); err != nil {
			state.RaiseError("headless browser: %s", err)
			return 0
		}
		track(browser)

		util.Push(state, browser, browserTypeName)
		return 1
	}
}

// launch starts a browser like rod does, with its download logged instead of printed on
// stdout, which must stay clean (e.g. --json).
func launch(ctx context.Context, log func(string, ...any), status func(context.Context, string)) (string, error) {
	l := launcher.New()
	if l.Get(flags.Bin) == "" {
		b := launcher.NewBrowser()
		b.Logger = logger(log)
		if b.Validate() != nil {
			status(ctx, "downloading chromium for the headless browser")
		}
		bin, err := b.Get()
		if err != nil {
			return "", fmt.Errorf("download browser: %w", err)
		}
		l = l.Bin(bin)
	}
	status(ctx, "starting the headless browser")
	return l.Launch()
}

type logger func(string, ...any)

func (l logger) Println(vs ...any) {
	l("%s", strings.TrimSpace(fmt.Sprintln(vs...)))
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
