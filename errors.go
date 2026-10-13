package luaprovider

import luahttp "github.com/luevano/luaprovider/lib/http"

// Errors to check with errors.Is (the same values as libmangal.ErrNotFound/ErrRateLimited), from what the scripts' requests answered (see the http
// lib's response:check()).
var (
	ErrNotFound    = luahttp.ErrNotFound
	ErrRateLimited = luahttp.ErrRateLimited
)

// HTTPError is a request that answered an error status other than 429, see the http lib.
type HTTPError = luahttp.Error

// RateLimitError is a request that answered 429.
type RateLimitError = luahttp.RateLimitError
