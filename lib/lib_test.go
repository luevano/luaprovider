package lib_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/luevano/luaprovider/lib"
	"github.com/luevano/luaprovider/util"
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
	err := util.GoError(run(t, `require("sdk").regexp.compile("(")`))
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
assert(table.concat(util.reverse(list), ",") == "2,1,3", "reverse")
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

func TestReplaceAllFuncPopsTheResults(t *testing.T) {
	script := `
local sdk = require("sdk")
local re = sdk.regexp.compile("a")
local out = re:replace_all_func(string.rep("a", 30000), function(m) return "b" end)
assert(out == string.rep("b", 30000), "replaced")
`
	if err := run(t, script); err != nil {
		t.Fatal(err)
	}
}

func TestUnsetOptionsUseTheDefaults(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, &lib.Options{})
	L.SetContext(context.Background())
	if err := L.DoString(`require("sdk").log.status("hello")`); err != nil {
		t.Fatal(err)
	}
}

func TestUtilBadArgumentsAreErrors(t *testing.T) {
	for _, code := range []string{
		`sdk.util.take({1, 2}, -1)`,
		`sdk.util.drop({1, 2}, -1)`,
		`sdk.util.drop_right({1, 2}, -1)`,
		`sdk.util.chunk({1, 2}, 0)`,
		`sdk.encoding.base64.encode("x", sdk.regexp.compile("a"))`,
	} {
		err := run(t, `local sdk = require("sdk"); `+code)
		if err == nil || !strings.Contains(err.Error(), "bad argument") {
			t.Errorf("%s: %v, want an argument error", code, err)
		}
	}

	script := `
local sdk = require("sdk")
assert(#sdk.util.take({1, 2}, 5) == 2, "take more than there is")
assert(#sdk.util.take({}, 1) == 0, "take from empty")
`
	if err := run(t, script); err != nil {
		t.Fatal(err)
	}
}

func TestJSON(t *testing.T) {
	script := `
local sdk = require("sdk")
local shared = {a = 1}
local out = sdk.encoding.json.encode({x = shared, y = shared})
assert(out == '{"x":{"a":1},"y":{"a":1}}', "shared table: " .. out)

local loop = {}
loop.self = loop
assert(not pcall(sdk.encoding.json.encode, loop), "a table inside itself")

for _ = 1, 20 do
	local keys = {}
	for k in pairs(sdk.encoding.json.decode('{"d": 1, "b": 2, "c": 3, "a": 4}')) do
		table.insert(keys, k)
	end
	assert(table.concat(keys) == "abcd", "keys in order: " .. table.concat(keys))
end
`
	if err := run(t, script); err != nil {
		t.Fatal(err)
	}
}

func TestLibErrorsKeepTheirChain(t *testing.T) {
	var syntaxErr *json.SyntaxError
	err := util.GoError(run(t, `require("sdk").encoding.json.decode("{")`))
	if !errors.As(err, &syntaxErr) {
		t.Errorf("json decode: wanted a *json.SyntaxError in the chain, got %v", err)
	}

	var urlErr *url.Error
	err = util.GoError(run(t, `require("sdk").urls.parse(":bad")`))
	if !errors.As(err, &urlErr) {
		t.Errorf("urls parse: wanted a *url.Error in the chain, got %v", err)
	}

	// scripts can still use the message
	if err := run(t, `
local ok, err = pcall(require("sdk").encoding.json.decode, "{")
assert(not ok)
assert(tostring(err):find("json decode:"), "tostring")
assert(("x: " .. err):find("json decode:"), "concat")
`); err != nil {
		t.Fatal(err)
	}
}

func TestJSSetOnlyTakesPlainValues(t *testing.T) {
	err := run(t, `
local sdk = require("sdk")
local vm = sdk.js.vm()
vm:set("t", {a = 1, b = {c = 2}})
assert(vm:run("t.a + t.b.c"):to_string() == "3", "plain table")
`)
	if err != nil {
		t.Fatal(err)
	}

	for name, value := range map[string]string{
		"userdata":        `sdk.http.request("GET", "http://localhost")`,
		"nested userdata": `{req = sdk.http.request("GET", "http://localhost")}`,
	} {
		err := run(t, `local sdk = require("sdk"); sdk.js.vm():set("v", `+value+`)`)
		if err == nil || !strings.Contains(util.GoError(err).Error(), "js set") {
			t.Errorf("%s: wanted a js set error, got %v", name, err)
		}
	}
}

func TestHTMLCallbackErrors(t *testing.T) {
	for _, method := range []string{"each", "map"} {
		err := util.GoError(run(t, `
local sdk = require("sdk")
sdk.html.parse("<p>a</p>"):find("p"):`+method+`(function() sdk.encoding.json.decode("{") end)
`))
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) || !strings.Contains(err.Error(), "html "+method+": json decode:") {
			t.Errorf("%s: wanted the callback's json error, got %v", method, err)
		}

		err = util.GoError(run(t, `require("sdk").html.parse("<p>a</p>"):find("p"):`+method+`(function() error("boom") end)`))
		if err == nil || !strings.Contains(err.Error(), "html "+method+":") || !strings.Contains(err.Error(), "boom") {
			t.Errorf("%s: wanted the callback's error, got %v", method, err)
		}
	}
}

func TestJSONValues(t *testing.T) {
	err := run(t, `
local json = require("sdk").encoding.json
local v = json.decode('{"list": [1, "two", true, {"k": 1.5}], "n": null, "s": "ñ"}')
assert(v.list[1] == 1 and v.list[2] == "two" and v.list[3] == true, "list")
assert(v.list[4].k == 1.5 and v.n == nil and v.s == "ñ", "nested")
assert(json.encode({1, 2, "x"}) == '[1,2,"x"]', "array: " .. json.encode({1, 2, "x"}))
assert(json.encode({a = {b = false}}) == '{"a":{"b":false}}', "object")
assert(not pcall(json.encode, {f = print}), "functions aren't json")
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBase64(t *testing.T) {
	err := run(t, `
local b = require("sdk").encoding.base64
assert(b.encode("hi?>") == "aGk/Pg==", "std")
assert(b.encode("hi?>", b.raw_url_encoding) == "aGk_Pg", "raw url")
assert(b.decode("aGk/Pg==") == "hi?>", "decode std")
assert(b.decode("aGk_Pg", b.raw_url_encoding) == "hi?>", "decode raw url")
assert(not pcall(b.decode, "@@@"), "invalid input")
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStrings(t *testing.T) {
	err := run(t, `
local s = require("sdk").strings
assert(s.trim_space("  a b  ") == "a b", "trim_space")
assert(s.replace_all("a-b-c", "-", "+") == "a+b+c", "replace_all")
assert(s.replace("a-b-c", "-", "+", 1) == "a+b-c", "replace")
local parts = s.split("a,b,c", ",")
assert(#parts == 3 and parts[3] == "c", "split")
assert(s.join({"a", "b"}, "/") == "a/b", "join")
assert(s.has_prefix("chapter 1", "chapter") and s.has_suffix("x.png", ".png"), "prefix/suffix")
assert(s.trim_prefix("/manga/1", "/manga/") == "1", "trim_prefix")
assert(s.contains("abc", "b") and not s.contains("abc", "d"), "contains")
assert(s.to_lower("AbC") == "abc" and s.to_upper("aBc") == "ABC", "case")
assert(s.count("cheese", "e") == 3 and s.index("chicken", "ken") == 4, "count/index")
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestURLs(t *testing.T) {
	err := run(t, `
local urls = require("sdk").urls
local u = urls.parse("https://example.com/a/b?x=1")
assert(u:hostname() == "example.com", "hostname")
assert(u:query():get("x") == "1", "query")
assert(u:join_path("c", "d"):string() == "https://example.com/a/b/c/d?x=1", "join_path: " .. u:join_path("c", "d"):string())
assert(u:parse("../z"):string() == "https://example.com/z", "relative: " .. u:parse("../z"):string())

local v = urls.values()
v:set("q", "a b")
v:add("t", "1")
assert(v:string() == "q=a+b&t=1", "values: " .. v:string())
assert(urls.path_escape("a b/c") == "a%20b%2Fc" and urls.path_unescape("a%20b") == "a b", "path escape")
assert(urls.query_escape("a b&") == "a+b%26" and urls.query_unescape("a+b%26") == "a b&", "query escape")
assert(not pcall(urls.path_unescape, "%zz"), "bad escape")
`)
	if err != nil {
		t.Fatal(err)
	}
}
