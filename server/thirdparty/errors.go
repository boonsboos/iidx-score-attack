package thirdparty

// TODO:
// implement custom error types
// this way we can differentiate between errors that are caused by the user (e.g. invalid refresh token)

type RefreshTokenError struct{}

func (e *RefreshTokenError) Error() string {
	return "Refresh token is invalid or expired"
}

type UnauthorizedError struct{}

func (e *UnauthorizedError) Error() string {
	return "Unauthorized"
}
