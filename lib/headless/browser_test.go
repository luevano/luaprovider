package headless

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

func TestBrowserCloseCleansUp(t *testing.T) {
	if testing.Short() || launcher.NewBrowser().Validate() != nil {
		t.Skip("needs rod's chromium already downloaded")
	}

	var tracked []io.Closer
	L := lua.NewState()
	defer L.Close()
	L.SetGlobal("headless", Lib(func(c io.Closer) { tracked = append(tracked, c) }, t.Logf, func(context.Context, string) {}).Value(L))
	L.SetContext(context.Background())
	if err := L.DoString(`browser = headless.browser()`); err != nil {
		t.Fatal(err)
	}
	b := L.GetGlobal("browser").(*lua.LUserData).Value.(*browser)
	dir := b.launcher.Get(flags.UserDataDir)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("user data dir: %v", err)
	}
	if len(tracked) != 1 || tracked[0] != io.Closer(b) {
		t.Fatalf("the browser should be tracked, got %v", tracked)
	}

	if err := L.DoString(`browser:close()`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("user data dir %s left behind: %v", dir, err)
	}
	// closed again with the provider
	if err := tracked[0].Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

func TestPages(t *testing.T) {
	if testing.Short() || launcher.NewBrowser().Validate() != nil {
		t.Skip("needs rod's chromium already downloaded")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/slow" {
			<-r.Context().Done()
			return
		}
		if r.URL.Path == "/other" {
			fmt.Fprint(w, `<p id="other">other page</p>`)
			return
		}
		fmt.Fprint(w, `<input id="in"><button id="b" onclick="document.getElementById('out').textContent = document.getElementById('in').value">go</button><p id="out"></p>`)
	}))
	defer server.Close()

	L := lua.NewState()
	defer L.Close()
	var tracked []io.Closer
	defer func() {
		for _, c := range tracked {
			c.Close()
		}
	}()
	L.SetGlobal("headless", Lib(func(c io.Closer) { tracked = append(tracked, c) }, t.Logf, func(context.Context, string) {}).Value(L))
	L.SetContext(context.Background())
	err := L.DoString(fmt.Sprintf(`
browser = headless.browser()
local page = browser:page(%[1]q)
page:element("#in"):input("typed")
page:element("#b"):click()
assert(page:element("#out"):html() == '<p id="out">typed</p>', "click: " .. page:element("#out"):html())
assert(page:element("#missing") == nil, "missing element")
page:navigate(%[1]q .. "/other")
assert(page:html():find("other page"), "navigate")
page:close()
`, server.URL))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(200*time.Millisecond, cancel)
	L.SetContext(ctx)
	err = util.GoError(L.DoString(fmt.Sprintf(`browser:page(%q)`, server.URL+"/slow")))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled page: wanted context.Canceled in the chain, got %v", err)
	}
}
