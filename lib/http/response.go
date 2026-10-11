package http

import (
	"bytes"
	"io"
	"net/http"

	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const responseTypeName = libName + "_response"

func responseStatus(L *lua.LState) int {
	response := util.Check[*http.Response](L, 1)
	L.Push(lua.LNumber(response.StatusCode))
	return 1
}

// responseCheck raises an error for a non-2xx status, matching ErrNotFound for 404.
func responseCheck(L *lua.LState) int {
	response := util.Check[*http.Response](L, 1)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		util.Raise(L, newError(response.Request, response))
		return 0
	}
	L.Push(L.Get(1))
	return 1
}

func responseBody(L *lua.LState) int {
	response := util.Check[*http.Response](L, 1)

	var (
		buffer []byte
		err    error
	)

	buffer, err = io.ReadAll(response.Body)
	if err != nil {
		L.RaiseError("http response body: %s", err)
		return 0
	}
	// so it can be read again
	response.Body = io.NopCloser(bytes.NewReader(buffer))

	L.Push(lua.LString(buffer))
	return 1
}

func responseHeader(L *lua.LState) int {
	response := util.Check[*http.Response](L, 1)
	key := L.CheckString(2)

	L.Push(lua.LString(response.Header.Get(key)))
	return 1
}

func responseCookies(L *lua.LState) int {
	response := util.Check[*http.Response](L, 1)

	cookies := L.NewTable()
	for _, cookie := range response.Cookies() {
		c := L.NewTable()
		c.RawSetString("name", lua.LString(cookie.Name))
		c.RawSetString("value", lua.LString(cookie.Value))

		cookies.Append(c)
	}

	L.Push(cookies)
	return 1
}

func responseContentLength(L *lua.LState) int {
	response := util.Check[*http.Response](L, 1)
	L.Push(lua.LNumber(response.ContentLength))
	return 1
}
