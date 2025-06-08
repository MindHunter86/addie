package utils

import "context"

// type ContextKey uint8

const (
	CtxZeroLogger ContextKey = iota
	CtxSyslogLogger
	CtxCliContext
	CtxAbortFunc
	CtxKeychainKeeper
	CtxDatabase
	CtxAuth0Service
)

func ContextValueExtract[V comparable](c context.Context, k ContextKey) V {
	v := c.Value(k).(V)
	return v
}
