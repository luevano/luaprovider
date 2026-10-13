package js

import (
	"context"
	"errors"
	"fmt"

	"github.com/luevano/luaprovider/util"
	"github.com/robertkrimen/otto"
	lua "github.com/yuin/gopher-lua"
)

const vmTypeName = libName + "_vm"

var errInterrupted = errors.New("interrupted")

func newVM(L *lua.LState) int {
	util.Push(L, otto.New(), vmTypeName)
	return 1
}

func vmRun(L *lua.LState) int {
	vm := util.Check[*otto.Otto](L, 1)
	script := L.CheckString(2)

	value, err := runInterruptible(L.Context(), vm, script)
	if err != nil {
		util.Raise(L, fmt.Errorf("js run: %w", err))
		return 0
	}

	util.Push(L, &value, valueTypeName)
	return 1
}

// runInterruptible: Stops the script when ctx is done, else a runaway script hangs forever.
func runInterruptible(ctx context.Context, vm *otto.Otto, script string) (value otto.Value, err error) {
	if ctx == nil {
		return vm.Run(script)
	}

	done := make(chan struct{})
	defer close(done)
	vm.Interrupt = make(chan func(), 1)
	go func() {
		select {
		case <-ctx.Done():
			vm.Interrupt <- func() { panic(errInterrupted) }
		case <-done:
		}
	}()

	// otto stops the script by panicking with what the interrupt panics with
	defer func() {
		if r := recover(); r != nil {
			if r != errInterrupted {
				panic(r)
			}
			err = ctx.Err()
		}
	}()
	return vm.Run(script)
}

func vmGet(L *lua.LState) int {
	vm := util.Check[*otto.Otto](L, 1)
	name := L.CheckString(2)

	value, err := vm.Get(name)
	if err != nil {
		util.Raise(L, fmt.Errorf("js get %q: %w", name, err))
		return 0
	}

	util.Push(L, &value, valueTypeName)
	return 1
}

func vmSet(L *lua.LState) int {
	vm := util.Check[*otto.Otto](L, 1)
	name := L.CheckString(2)
	lvalue := L.CheckAny(3)

	value, err := util.FromLValue(lvalue)
	if err != nil {
		util.Raise(L, fmt.Errorf("js set %q: %w", name, err))
		return 0
	}

	err = vm.Set(name, value)
	if err != nil {
		util.Raise(L, fmt.Errorf("js set %q: %w", name, err))
		return 0
	}

	return 0
}
