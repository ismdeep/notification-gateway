package rest

import "errors"

var (
	ErrUnauthorized    = errors.New("ERROR_UNAUTHORIZED")
	ErrRequestBindJSON = errors.New("ERROR_REQUEST_BIND_JSON")
)
