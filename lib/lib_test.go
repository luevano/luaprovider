package lib_test

import (
	"context"
	"strings"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

func run(t *testing.T, script string) error {
	t.Helper()
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())
	return L.DoString(script)
}

func TestHTMLAndRegexp(t *testing.T) {
	script := `
local sdk = require("sdk")
local doc = sdk.html.parse("<ul><li>a</li><li>b</li></ul>")
local items = {}
doc:find("li"):each(function(s) table.insert(items, s:text()) end)
assert(table.concat(items) == "ab", "each")
local texts = doc:find("li"):map(function(s) return s:text() end)
assert(#texts == 2 and texts[2] == "b", "map")
assert(doc:find("ul"):first():html() == "<li>a</li><li>b</li>", "html")

local re = sdk.regexp.compile("(?P<n>[0-9]+)")
assert(re:groups("ch 12").n == "12", "groups")
assert(sdk.regexp.urls_strict:match("see https://example.com"), "vars")
`
	if err := run(t, script); err != nil {
		t.Fatal(err)
	}
}

func TestLibErrorsSayWhatFailed(t *testing.T) {
	err := run(t, `require("sdk").regexp.compile("(")`)
	if err == nil || !strings.Contains(err.Error(), "regexp compile:") {
		t.Fatalf("wanted a regexp compile error, got %v", err)
	}
}
