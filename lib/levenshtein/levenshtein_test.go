package levenshtein

import (
	"sync"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func call(t *testing.T, s1, s2 string) int {
	t.Helper()
	L := lua.NewState()
	defer L.Close()
	fn := L.NewFunction(distance)
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, lua.LString(s1), lua.LString(s2)); err != nil {
		t.Fatalf("distance(%q, %q): %v", s1, s2, err)
	}
	return int(L.CheckNumber(-1))
}

func TestDistance(t *testing.T) {
	for _, c := range []struct {
		s1, s2 string
		want   int
	}{
		{"kitten", "sitting", 3},
		{"", "", 0},
		{"One Piece", "one piece", 2},
		{"ワンピース", "ワンピス", 1},
		{"ñ", "", 1},
		{"", "日本", 2},
		{"𝔸b", "ab", 1},
		{"𝔸", "𝔹", 1},
	} {
		if got := call(t, c.s1, c.s2); got != c.want {
			t.Errorf("distance(%q, %q) = %d, want %d", c.s1, c.s2, got, c.want)
		}
	}
}

func TestDistanceConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			L := lua.NewState()
			defer L.Close()
			fn := L.NewFunction(distance)
			for range 2000 {
				if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, lua.LString("kitten"), lua.LString("sitting")); err != nil {
					t.Error(err)
					return
				}
				if d := L.CheckNumber(-1); d != 3 {
					t.Errorf("distance = %v, want 3", d)
					return
				}
				L.Pop(1)
			}
		})
	}
	wg.Wait()
}
