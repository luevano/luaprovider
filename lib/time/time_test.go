package time_test

import (
	"context"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

func TestTimeMethods(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	script := `
local time = require("sdk").time
local t = time.parse("2006-01-02", "2024-03-05")
assert(t:year() == 2024 and t:month() == 3 and t:day() == 5, "date")
assert(t:format("02/01/2006") == "05/03/2024", "format")
local later = t:add(time.hour)
assert(later:after(t) and t:before(later), "compare")
assert(later:sub(t) == time.hour, "sub")
assert(time.unix(0, 0):equal(time.unix(0, 0)), "unix")
assert(not pcall(time.parse, "2006-01-02", "nope"), "parse error")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}
