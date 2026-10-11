package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Errors to check with errors.Is.
var (
	ErrNotFound    = errors.New("not found")
	ErrRateLimited = errors.New("rate limited")
)

// Error is a response with an error status: raised by response:check(), and by send for
// 429 (no script wants to handle a rate limit itself).
type Error struct {
	Method, URL string
	StatusCode  int
	// RetryAfter is the Retry-After of a 429, zero if not sent.
	RetryAfter time.Duration
}

func newError(request *http.Request, response *http.Response) *Error {
	e := &Error{Method: request.Method, URL: request.URL.String(), StatusCode: response.StatusCode}
	if response.StatusCode == http.StatusTooManyRequests {
		e.RetryAfter = retryAfter(response.Header.Get("Retry-After"))
	}
	return e
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s: %d %s", e.Method, e.URL, e.StatusCode, strings.ToLower(http.StatusText(e.StatusCode)))
}

func (e *Error) Is(target error) bool {
	switch e.StatusCode {
	case http.StatusNotFound:
		return target == ErrNotFound
	case http.StatusTooManyRequests:
		return target == ErrRateLimited
	}
	return false
}

// RetryAfterDuration is how long to wait before retrying a 429.
func (e *Error) RetryAfterDuration() time.Duration {
	return e.RetryAfter
}

// retryAfter parses a Retry-After header: seconds or an http date.
func retryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		return max(time.Until(t), 0)
	}
	return 0
}
