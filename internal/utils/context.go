package utils

import "context"

type ContextKey uint16

const (
	CtxZeroLogger ContextKey = iota
	CtxCliContext
	CtxAccsLogger
	CtxStats

	CtxController

	CtxBalancers
	CtxTitleCache
	CtxBlockList

	CtxRuntime
	CtxRPatcher

	CtxDatabase

	// CtxAbortFunc

	// internal use only
	CtxDynamic
	CtxTickerLap
	CtxTickerTick
)

var CKtoa = map[ContextKey]string{
	CtxZeroLogger: "system logger",
	CtxCliContext: "cli context",
	CtxAccsLogger: "access logger",
	CtxStats:      "stats",

	CtxBalancers:  "balancers",
	CtxTitleCache: "titles cache",
	CtxBlockList:  "blocklist",

	CtxRuntime:  "runtime",
	CtxRPatcher: "request patcher",

	CtxDatabase: "bbolt database",

	// internal use only
	CtxDynamic:    "",
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
