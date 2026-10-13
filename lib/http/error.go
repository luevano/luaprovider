package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/luevano/libmangal/netutil"
)

// Errors to check with errors.Is, the same values as libmangal's.
var (
	ErrNotFound    = netutil.ErrNotFound
	ErrRateLimited = netutil.ErrRateLimited
)

// Error is a response with an error status other than 429 (see RateLimitError), raised by
// response:check().
type Error struct {
	Method, URL string
	StatusCode  int
}

// NewError is the error for a response with an error status: a *RateLimitError for 429, else
// an *Error.
func NewError(request *http.Request, response *http.Response) error {
	if response.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{
			Method:     request.Method,
			URL:        request.URL.String(),
			RetryAfter: netutil.RetryAfter(response.Header.Get("Retry-After")),
		}
	}
	return &Error{Method: request.Method, URL: request.URL.String(), StatusCode: response.StatusCode}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s: %d %s", e.Method, e.URL, e.StatusCode, strings.ToLower(http.StatusText(e.StatusCode)))
}

func (e *Error) Is(target error) bool {
	return e.StatusCode == http.StatusNotFound && target == ErrNotFound
}

// RateLimitError is a 429 response, raised by send (no script wants to handle a rate limit
// itself) and response:check().
type RateLimitError struct {
	Method, URL string
	// RetryAfter is the response's Retry-After, zero if not sent.
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s %s: 429 too many requests", e.Method, e.URL)
}

func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimited
}

// RetryAfterDuration is how long to wait before retrying, zero if the server didn't say.
func (e *RateLimitError) RetryAfterDuration() time.Duration {
	return e.RetryAfter
}
