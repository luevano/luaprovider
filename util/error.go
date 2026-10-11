package util

import (
	"errors"

	lua "github.com/yuin/gopher-lua"
)

// Raise raises err as a lua error that keeps it: scripts see its message (tostring), and
// GoError gets it back from what the call returns.
func Raise(L *lua.LState, err error) {
	ud := L.NewUserData()
	ud.Value = err
	mt := L.NewTable()
	mt.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(err.Error()))
		return 1
	}))
	L.SetMetatable(ud, mt)
	L.Error(ud, 1)
}

// GoError is the Go error a script raised with Raise (also re-raised by the script with
// error(e)), else err.
func GoError(err error) error {
	var apiErr *lua.ApiError
	if !errors.As(err, &apiErr) {
		return err
	}
	if ud, ok := apiErr.Object.(*lua.LUserData); ok {
		if goErr, ok := ud.Value.(error); ok {
			return goErr
		}
	}
	return err
}
