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

func TestUtil(t *testing.T) {
	script := `
local util = require("sdk").util
local list = {3, 1, 2}
local sorted = util.sort(list)
assert(table.concat(sorted, ",") == "1,2,3", "sort default")
sorted = util.sort(list, function(a, b) return a > b end)
assert(table.concat(sorted, ",") == "3,2,1", "sort custom")
assert(table.concat(util.sort({"b", "a"}), ",") == "a,b", "sort strings")
assert(util.some_by(list, function(v) return v == 2 end), "some_by")
assert(not util.every_by(list, function(v) return v > 1 end), "every_by")
local mapped = util.map_async({a = 1, b = 2}, function(v) return v * 2 end)
assert(mapped.a == 2 and mapped.b == 4, "map_async")
assert(#util.filter(list, function(v) return v > 1 end) == 2, "filter")
assert(util.reduce(list, function(acc, v) return acc + v end, 0) == 6, "reduce")
assert(#util.shuffle(list) == 3 and #util.samples(list, 2) == 2, "samples")
`
	if err := run(t, script); err != nil {
		t.Fatal(err)
	}
}

func TestUtilCallbacksDontGrowTheStack(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	err := L.DoString(`
local util = require("sdk").util
local list = {}
for i = 1, 100000 do list[i] = i end
util.for_each(list, function(v) end)
assert(#util.filter(list, function(v) return true end) == 100000, "filter")
`)
	if err != nil {
		t.Fatal(err)
	}
}
