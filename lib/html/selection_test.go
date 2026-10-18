package html_test

import (
	"context"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

func TestSliceClamps(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	script := `
local items = require("sdk").html.parse("<i>1</i><i>2</i><i>3</i><i>4</i><i>5</i>"):find("i")
local function texts(s)
	local out = {}
	s:each(function(i) out[#out + 1] = i:text() end)
	return table.concat(out)
end
assert(texts(items:slice(0, 20)) == "12345", "past the end")
assert(texts(items:slice(1, 3)) == "23", "inside")
assert(texts(items:slice(-2, 5)) == "45", "negative start")
assert(texts(items:slice(-20, 2)) == "12", "negative start past the beginning")
assert(texts(items:slice(0, -1)) == "1234", "negative end")
assert(texts(items:slice(4, 2)) == "", "end before start")
assert(texts(items:slice(10, 20)) == "", "start past the end")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}
