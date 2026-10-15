package headless

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const browserTypeName = libName + "_browser"

// browser is a launched browser, closed by the script or with the provider.
type browser struct {
	*rod.Browser
	launcher  *launcher.Launcher
	closeOnce sync.Once
	closeErr  error
}

// Close closes the browser, kills it if it doesn't exit and removes its user data dir.
func (b *browser) Close() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.Browser.Close()
		if b.closeErr != nil {
			b.launcher.Kill()
		}
		cleanup(b.launcher)
	})
	return b.closeErr
}

// cleanup waits for the browser to exit (killed if it takes too long) and removes its user
// data dir.
func cleanup(l *launcher.Launcher) {
	exited := make(chan struct{})
	go func() {
		l.Cleanup()
		close(exited)
	}()
	select {
	case <-exited:
		return
	case <-time.After(10 * time.Second):
	}
	l.Kill()
	select {
	case <-exited:
	// rod's exit signal can be missing (e.g. leakless failed), the dir is all that's left
	case <-time.After(5 * time.Second):
		_ = os.RemoveAll(l.Get(flags.UserDataDir))
	}
}

// newNewBrowser: Browsers are tracked so they're closed with the provider.
func newNewBrowser(track func(io.Closer), log func(string, ...any), status func(context.Context, string)) lua.LGFunction {
	return func(state *lua.LState) int {
		l, u, err := launch(state.Context(), log, status)
		if err != nil {
			util.Raise(state, fmt.Errorf("headless browser: %w", err))
			return 0
		}
		// the browser outlives the call, its connection doesn't take the call's ctx
		b := &browser{Browser: rod.New().ControlURL(u), launcher: l}
		if err := b.Connect(); err != nil {
			l.Kill()
			cleanup(l)
			util.Raise(state, fmt.Errorf("headless browser: %w", err))
			return 0
		}
		track(b)

		util.Push(state, b, browserTypeName)
		return 1
	}
}

// launch starts a browser like rod does, with its download logged instead of printed on
// stdout, which must stay clean (e.g. --json). It and cleanup are duplicated in
// mangoprovider's scraper/headless/rod (no shared module but libmangal, which shouldn't
// pull rod): keep both in sync.
func launch(ctx context.Context, log func(string, ...any), status func(context.Context, string)) (*launcher.Launcher, string, error) {
	// leakless (kills chromium if mangal dies) only on linux: antiviruses flag its
	// binary on windows, same as mangoprovider did
	l := launcher.New().Leakless(runtime.GOOS == "linux").Context(ctx)
	if l.Get(flags.Bin) == "" {
		b := launcher.NewBrowser()
		b.Context = ctx
		b.Logger = logger(log)
		if b.Validate() != nil {
			status(ctx, "downloading chromium for the headless browser")
		}
		bin, err := b.Get()
		if err != nil {
			return nil, "", fmt.Errorf("download browser: %w", err)
		}
		l = l.Bin(bin)
	}
	status(ctx, "starting the headless browser")
	u, err := l.Launch()
	if err != nil {
		// a browser that started but didn't answer is killed by Launch
		if l.PID() != 0 {
			cleanup(l)
		}
		return nil, "", err
	}
	return l, u, nil
}

type logger func(string, ...any)

func (l logger) Println(vs ...any) {
	l("%s", strings.TrimSpace(fmt.Sprintln(vs...)))
}

func browserPage(state *lua.LState) int {
	browser := util.Check[*browser](state, 1)
	URL := state.CheckString(2)

	// the browser can outlive a call, each operation uses the context of the current one
	page, err := browser.Context(state.Context()).Page(proto.TargetCreateTarget{URL: URL})
	if err != nil {
		util.Raise(state, fmt.Errorf("headless page %s: %w", URL, err))
		return 0
	}
	if err := page.WaitLoad(); err != nil {
		page.Close()
		util.Raise(state, fmt.Errorf("headless page %s: %w", URL, err))
		return 0
	}

	util.Push(state, page, pageTypeName)
	return 1
}

func browserClose(state *lua.LState) int {
	browser := util.Check[*browser](state, 1)
	if err := browser.Close(); err != nil {
		util.Raise(state, fmt.Errorf("headless browser close: %w", err))
	}
	return 0
}
