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
			RetryAfter: retryAfter(response.Header.Get("Retry-After")),
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
