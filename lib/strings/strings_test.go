package strings_test

import (
	"context"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

func TestCountAny(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	script := `
local strings = require("sdk").strings
assert(strings.count_any("hello world", "lo") == 5, "runes, not substring")
assert(strings.count_any("hello", "xyz") == 0, "none")
assert(strings.count_any("hello", "") == 0, "empty chars")
assert(strings.count_any("", "abc") == 0, "empty s")
assert(strings.count_any("ñandú ñu", "ñú") == 3, "unicode")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func TestTitle(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	script := `
local strings = require("sdk").strings
assert(strings.title("hello world") == "Hello World", "words")
assert(strings.title("one-piece vol.1") == "One-Piece Vol.1", "separators")
assert(strings.title("hELLO") == "HELLO", "rest of the word untouched")
assert(strings.title("élan ñandú") == "Élan Ñandú", "unicode")
assert(strings.title("") == "", "empty")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicate(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())

	script := `
local strings = require("sdk").strings
assert(strings.duplicate("ab", 3) == "ababab", "repeat")
assert(strings.duplicate("ab", 0) == "", "zero")
assert(not pcall(strings.duplicate, "ab", -1), "negative")
assert(not pcall(strings.duplicate, "ab", 2^31), "huge")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}
