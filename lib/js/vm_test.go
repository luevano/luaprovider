package js_test

import (
	"context"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

func TestSetTableCycle(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	script := `
local vm = require("sdk").js.vm()
local t = {}
t.self = t
local ok, err = pcall(vm.set, vm, "t", t)
assert(not ok, "a table inside itself is an error")
assert(tostring(err):find("inside itself"), tostring(err))

local shared = {n = 1}
vm:set("s", {a = shared, b = shared})
assert(vm:run("s.a.n + s.b.n"):export() == 2, "a table twice in siblings is fine")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}
