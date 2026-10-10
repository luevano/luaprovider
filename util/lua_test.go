package util

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestFromLTableReportsErrors(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	table := L.NewTable()
	table.RawSetString("ok", lua.LString("value"))
	table.RawSetString("fn", L.NewFunction(func(*lua.LState) int { return 0 }))

	if _, err := FromLTable(table); err == nil {
		t.Error("a function can't be converted, wanted an error")
	}

	table.RawSetString("fn", lua.LNil)
	result, err := FromLTable(table)
	if err != nil || result["ok"] != "value" {
		t.Errorf("wanted the converted table, got %v, %v", result, err)
	}
}
