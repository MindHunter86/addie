package utils

import "context"

type ContextKey uint16

const (
	CtxZeroLogger ContextKey = iota
	CtxCliContext
	CtxAccsLogger

	CtxChallenge
	CtxWebRender
	CtxDatabase
	CtxStats
	CtxAnalytics

	// internal use only
	CtxDynamic

	CtxTickerLap
	CtxTickerTick
)

var CKtoa = map[ContextKey]string{
	CtxZeroLogger: "system logger",
	CtxCliContext: "cli context",
	CtxAccsLogger: "access logger",

	CtxChallenge: "challenger",
	CtxWebRender: "webrender",
	CtxDatabase:  "database",
	CtxStats:     "stats",
	CtxAnalytics: "analytics",

	// internal use only
	CtxTickerLap:  "",
	CtxTickerTick: "",
}

func ContextValueExtract[V comparable](c context.Context, k ContextKey) (v V) {
	if c.Value(k) == nil {
		return
	}

	v = c.Value(k).(V)
	return v
}
