package http

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/luevano/libmangal/netutil"
	"github.com/luevano/luaprovider/util"
	"github.com/philippgille/gokv"
	lua "github.com/yuin/gopher-lua"
)

const requestTypeName = libName + "_request"

// newRequestNew: New requests get the user agent as default, scripts can still change it.
func newRequestNew(userAgent string) lua.LGFunction {
	return func(L *lua.LState) int {
		method := L.CheckString(1)
		url := L.CheckString(2)
		body := L.OptString(3, "")

		request, err := http.NewRequestWithContext(
			L.Context(),
			method,
			url,
			strings.NewReader(body),
		)
		if err != nil {
			util.Raise(L, fmt.Errorf("http request %s %s: %w", method, url, err))
		}
		if userAgent != "" {
			request.Header.Set("User-Agent", userAgent)
		}

		util.Push(L, request, requestTypeName)
		return 1
	}
}

func requestHeader(L *lua.LState) int {
	request := util.Check[*http.Request](L, 1)
	key := L.CheckString(2)

	if L.GetTop() == 3 {
		value := L.CheckString(3)
		request.Header.Set(key, value)
		return 0
	}

	L.Push(lua.LString(request.Header.Get(key)))
	return 1
}

func requestCookie(L *lua.LState) int {
	request := util.Check[*http.Request](L, 1)
	key := L.CheckString(2)

	if L.GetTop() == 3 {
		value := L.CheckString(3)
		cookie := &http.Cookie{Name: key, Value: value}
		request.AddCookie(cookie)
		return 0
	}

	cookie, _ := request.Cookie(key)
	if cookie == nil {
		L.Push(lua.LNil)
		return 1
	}

	L.Push(lua.LString(cookie.Value))
	return 1
}

func requestContentLength(L *lua.LState) int {
	request := util.Check[*http.Request](L, 1)

	if L.GetTop() == 2 {
		value := L.CheckInt64(2)
		request.ContentLength = value
		return 0
	}

	L.Push(lua.LNumber(request.ContentLength))
	return 1
}

func requestSend(L *lua.LState, client *http.Client, store gokv.Store, flare *flareSolverr, debug func(string, ...any)) int {
	request := util.Check[*http.Request](L, 1)

	key, cacheable := cacheKey(request)
	if cacheable {
		if response, found := cachedResponse(store, key, request); found {
			if debug != nil {
				debug("%s %s cached", request.Method, netutil.RedactURL(request.URL))
			}
			util.Push(L, response, responseTypeName)
			return 1
		}
	}
	if viaFlareSolverr(request) {
		client = flare.get()
	}

	response, err := client.Do(request)
	if err != nil {
		util.Raise(L, fmt.Errorf("http send %s %s: %w", request.Method, request.URL, err))
		return 0
	}
	// read it all so the connection is released, body() returns it from memory
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		util.Raise(L, fmt.Errorf("http send %s %s: read body: %w", request.Method, request.URL, err))
		return 0
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	if response.StatusCode == http.StatusTooManyRequests {
		util.Raise(L, NewError(request, response))
		return 0
	}

	if cacheable && response.StatusCode == http.StatusOK {
		cacheResponse(store, key, response)
	}

	util.Push(L, response, responseTypeName)
	return 1
}

// cacheKey: Only GET and HEAD requests are cached. The key is hashed so cookies and auth headers
// aren't stored as they are, requests with different cookies still get different keys.
func cacheKey(request *http.Request) (string, bool) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return "", false
	}
	dump, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(dump)
	return hex.EncodeToString(sum[:]), true
}

func cachedResponse(store gokv.Store, key string, request *http.Request) (*http.Response, bool) {
	var dump []byte
	found, err := store.Get(key, &dump)
	if err != nil || !found {
		return nil, false
	}
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(dump)), request)
	if err != nil {
		_ = store.Delete(key)
		return nil, false
	}
	return response, true
}

// cacheResponse: Stores the response without the cookies it sets.
func cacheResponse(store gokv.Store, key string, response *http.Response) {
	cookies := response.Header.Values("Set-Cookie")
	response.Header.Del("Set-Cookie")
	// DumpResponse puts the body back after reading it
	dump, err := httputil.DumpResponse(response, true)
	for _, cookie := range cookies {
		response.Header.Add("Set-Cookie", cookie)
	}
	if err == nil {
		_ = store.Set(key, dump)
	}
}
