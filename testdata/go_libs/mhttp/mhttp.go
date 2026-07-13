package mhttp

import "net/http"

type VersionedHandlers struct {
	V1 http.Handler
	V2 http.Handler
}

func NewVersionedHandlers(handlers ...any) VersionedHandlers {
	return VersionedHandlers{}
}
