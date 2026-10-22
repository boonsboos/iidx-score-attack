package thirdparty

import "strconv"

type RefreshTokenError struct{}

func (e *RefreshTokenError) Error() string {
	return "Refresh token is invalid or expired"
}

type UnauthorizedError struct{}

func (e *UnauthorizedError) Error() string {
	return "Unauthorized"
}

type FServerError struct{}

func (e *FServerError) Error() string {
	return "F server side error"
}

type RateLimitError struct {
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	return "Rate limit exceeded, retry after: " + strconv.Itoa(e.RetryAfter) + " seconds"
}
