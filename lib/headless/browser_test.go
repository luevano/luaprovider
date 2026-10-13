package headless

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
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
