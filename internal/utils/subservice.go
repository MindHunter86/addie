package utils

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/rs/zerolog"
)

type SubserviceHandler func(context.Context) (contextkey any, err error)

var subservices map[ContextKey]SubserviceHandler

func RegisterSubservice(ck ContextKey, fn SubserviceHandler) {
	if subservices == nil {
		subservices = make(map[ContextKey]SubserviceHandler)
	}

	subservices[ck] = fn
}

func ForEachSubservice(fn func(ContextKey, SubserviceHandler) error) (e error) {
	for k, service := range subservices {
		if e = fn(k, service); e != nil {
			return
		}
	}
	return
}

type CallbackEvent uint8

const (
	OnServiceBootstrap CallbackEvent = iota
	OnServiceTicker1sec
	OnServiceDestruct
)

type ServiceCallback func(context.Context) error

var callbacks map[CallbackEvent][]ServiceCallback

// Should be called during subservice registration or in init() functions
func RegisterCallback(e CallbackEvent, fn ServiceCallback) {
	if callbacks == nil {
		callbacks = make(map[CallbackEvent][]ServiceCallback)
	}

	if callbacks[e] == nil {
		callbacks[e] = make([]ServiceCallback, 0, 32)
	}

	callbacks[e] = append(callbacks[e], fn)
}

func CallCallbacks(e CallbackEvent, fn func(cb ServiceCallback) error) (err error) {
	if callbacks == nil || callbacks[e] == nil {
		return
	}

	for _, cb := range callbacks[e] {
		if err = fn(cb); err != nil {
			return
		}
	}

	return
}

func GoCallCallbacks(e CallbackEvent, wg *sync.WaitGroup, l *zerolog.Logger, fn func(cb ServiceCallback)) {
	if callbacks == nil || callbacks[e] == nil {
		return
	}

	for _, cb := range callbacks[e] {
		Go(wg, l, func() { fn(cb) })
	}
}

// goroutine helper
func Go(wg *sync.WaitGroup, l *zerolog.Logger, fn func()) {
	wg.Add(1)
	go func(done, payload func()) {
		defer done()
		defer func() {
			if r := recover(); r != nil {
				l.Error().Msgf("go subservice panic has been caught\n%s\n%s", r, debug.Stack())
			}
		}()
		payload()
	}(wg.Done, fn)
}
