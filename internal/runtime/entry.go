package runtime

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type entryValue uint8

const (
	entryCurrent   entryValue = iota
	entryCandidate            // entryCandidate - used as maxlen in make
)

type Entry struct {
	wg     sync.WaitGroup
	wginit sync.WaitGroup

	mu sync.RWMutex

	value map[entryValue]interface{}

	deployed   bool
	deployDone context.CancelFunc
	deployStep int
}

func newEntry(v interface{}) *Entry {
	val := make(map[entryValue]interface{}, entryCandidate)
	val[entryCurrent] = v

	return &Entry{
		value: val,

		deployed:   true,
		deployStep: -1,
	}
}

func (m *Entry) deploy(val interface{}) {
	log.Trace().Msgf("smoothly deploy called with value %+v", val)

	// wait for deploy initialization before redeploying
	m.wginit.Wait()

	// candidate check for avoiding dummy "rejections"
	if val == m.candidate() {
		log.Debug().Msg("requested value is already patching now, reject new deploy request")
		return
	}

	// check if there is active deploy right now
	if !m.deployed {
		if m.deployDone == nil {
			panic("unhandled conditions")
		}

		log.Trace().Msg("concurrent deploy detected, send done() and wait for closing")
		m.deployDone()
		m.wg.Wait()
	}

	if !m.deployed {
		panic("unhandled conditions")
	}

	// disable further deploy for unchanged values
	if m.compare(val) {
		log.Trace().Msgf("given value already has been applied, deploy stopped")
		return
	}

	// block initial proccess
	m.wginit.Add(1)

	// syncing all goroutines in the deploy process
	m.wg.Add(1)
	defer m.wg.Done()

	// create cancel func for concurrent deploys
	var ctx context.Context
	ctx, m.deployDone = context.WithCancel(context.Background())

	// setup timer for step counter
	ticker := time.NewTicker(deployInteration)
	defer ticker.Stop()

	// set/reset step counter
	m.tick(true)
	defer m.execWithBlock(func() { m.deployStep, m.deployed = -1, true })

	// set candidate value
	m.execWithBlock(func() { m.prepare(val) })

	// unblock inital proccess and start event loop
	m.wginit.Done()

loop:
	for {
		select {
		case <-ticker.C:
			if m.tick() {
				break loop
			}
		case <-ctx.Done():
			log.Trace().Msg("ConfigDeploy - new value received for deploying; current deploy has been canceled")
			return
		case <-done():
			log.Trace().Msg("ConfigDeploy - internal abort() has been caught")
			return
		}
	}

	// finally apply/commit candidate value
	log.Debug().Msgf("new config value is commiting: %+v to %+v", m.get(true), m.candidate())
	m.set(m.candidate())
}

func (m *Entry) execWithBlock(exec func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec()
}

func (m *Entry) tick(initial ...bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	initial = append(initial, false)
	if initial[0] {
		log.Trace().Msgf("initial tick detected with step %d", deployStep)
		m.deployStep, m.deployed = deployStep, false
		return false
	}

	log.Trace().Msgf("ticked with step %d, updated %d", m.deployStep, m.deployStep-1)
	m.deployStep--
	return m.deployStep == 0
}

func (m *Entry) prepare(val interface{}) {
	m.value[entryCandidate] = val
	log.Trace().Msgf("candidate commited - %+v", val)
}

func (m *Entry) compare(val interface{}) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.value[entryCurrent] == val
}

func (m *Entry) candidate() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.value[entryCandidate]
}

func (m *Entry) get(force ...bool) interface{} {
	m.mu.RLock()
	if force = append(force, false); !m.deployed && !force[0] {
		m.mu.RUnlock()
		return m.randomize()
	}
	defer m.mu.RUnlock()

	return m.value[entryCurrent]
}

func (m *Entry) randomize() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// TODO: find more efficient/smoothly method for value randomizing
	if m.deployStep < rand.Intn(deployStep)+1 { // skipcq: GSC-G404 math/rand is enough
		return m.value[entryCandidate]
	}

	return m.value[entryCurrent]
}

func (m *Entry) set(val interface{}) {
	if m.compare(val) {
		log.Trace().Msgf("given value already has been applied, deploy stopped")
		return
	}

	m.execWithBlock(func() { m.value[entryCurrent], m.value[entryCandidate] = val, nil })
	log.Trace().Msgf("value commited - %+v", val)
}
