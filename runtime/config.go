package runtime

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/MindHunter86/addie/utils"
	"github.com/urfave/cli/v2"
)

type (
	ConfigEid   uint8
	ConfigParam uint8

	// ConfigEntry   map[ConfigEid]interface{}
	ConfigStorage map[ConfigParam]*ConfigEntry

	ConfigEntry struct {
		sync.RWMutex

		Payload interface{}
		Target  interface{}
		Step    int
	}
)

const (
	ConfigParamLottery ConfigParam = iota
	ConfigParamQuality
	ConfigParamBlocklist
	ConfigParamBlocklistIps
	ConfigParamLimiter
	ConfigParamA5bility
	ConfigParamStdoutAccess

	_configParamMaxSize
)

var ConfigParamDefaults = map[ConfigParam]interface{}{
	ConfigParamLottery:      100,
	ConfigParamQuality:      utils.TitleQualityHD,
	ConfigParamBlocklist:    1,
	ConfigParamBlocklistIps: []string{},
	ConfigParamLimiter:      0,
	ConfigParamA5bility:     0,
	ConfigParamStdoutAccess: 0,
}

var GetNameByConfigParam = map[ConfigParam]string{
	ConfigParamLottery:      "lottery",
	ConfigParamQuality:      "quality",
	ConfigParamBlocklist:    "blocklist",
	ConfigParamBlocklistIps: "blocklist_ips",
	ConfigParamLimiter:      "limiter",
	ConfigParamA5bility:     "cluster-availability",
	ConfigParamStdoutAccess: "stdout-access-log",
}

var sLocker sync.RWMutex

var deployStep int
var deployInteration time.Duration

var done func() <-chan struct{}

var (
	ErrConfigStorageLockFailure = errors.New("config storage - could not lock storage")
	ErrConfigEntryLockFailure   = errors.New("config storage - could not lock entry")
	ErrConfigInvalidParam       = errors.New("config storage - invalid param or internal map error")
	ErrConfigInvalidStep        = errors.New("config storage - softer-step must be >= 0")
)

func NewConfigStorage(c context.Context) (ConfigStorage, error) {
	done = c.Done

	ccx := c.Value(utils.ContextKeyCliContext).(*cli.Context)
	deployStep, deployInteration =
		ccx.Int("balancer-softer-step"),
		ccx.Duration("balancer-softer-tick")

	if deployStep < 0 {
		return nil, ErrConfigInvalidStep
	}

	if deployInteration < 10*time.Second {
		log.Warn().Msg("low value detected for softer-tick arg")
	}

	return make(ConfigStorage, _configParamMaxSize), nil
}

func (m ConfigStorage) GetValue(param ConfigParam) (val interface{}, ok bool, e error) {
	var entry *ConfigEntry
	if entry, ok, e = m.getEntry(param); e != nil || !ok {
		return
	}

	val, e = entry.Value()
	return
}

func (m ConfigStorage) SetValue(param ConfigParam, val interface{}) (e error) {
	var ok bool
	var entry *ConfigEntry

	if entry, ok, e = m.getEntry(param); e != nil {
		return
	} else if !ok {
		entry = newConfigEntry(param, val)
		return m.setEntry(param, entry)
	}

	var eval interface{}
	if eval, e = entry.Value(); e != nil {
		log.Debug().Msgf("could not get value for config-deploy: %s", e.Error())
		return
	} else if eval == val {
		log.Trace().Msg("config deploy will be skipped, because of no changes")
		return
	}

	return entry.SetValue(val)
}

func (m ConfigStorage) SetValueSmoothly(param ConfigParam, val interface{}) (e error) {
	var ok bool
	var entry *ConfigEntry

	if entry, ok, e = m.getEntry(param); e != nil {
		return
	} else if !ok {
		entry = newConfigEntry(param, nil)

		if e = m.setEntry(param, entry); e != nil {
			return
		}
	}

	var eval interface{}
	if eval, e = entry.Value(); e != nil {
		log.Debug().Msgf("could not get value for smooth-deploy: %s", e.Error())
		return
	} else if eval == val {
		log.Trace().Msg("smooth-deploy will be skipped, because of no changes")
		return
	}

	if e = entry.SetTarget(val); e != nil {
		return
	}

	if _, e = entry.nextDeployStep(true); e != nil {
		return
	}

	go entry.bootstrapDeploy()

	return
}

func (m ConfigStorage) getEntry(param ConfigParam) (entry *ConfigEntry, ok bool, e error) {
	if !sLocker.TryRLock() {
		e = ErrConfigStorageLockFailure
		return
	}
	defer sLocker.RUnlock()

	entry, ok = m[param]
	return
}

func (m ConfigStorage) setEntry(param ConfigParam, entry *ConfigEntry) (e error) {
	if !sLocker.TryLock() {
		e = ErrConfigStorageLockFailure
		return
	}
	defer sLocker.Unlock()

	m[param] = entry
	return
}

// ---

func newConfigEntry(param ConfigParam, value interface{}) *ConfigEntry {
	payload := ConfigParamDefaults[param]

	if value != nil {
		payload = value
	}

	return &ConfigEntry{
		Payload: payload,
		Step:    -1,
	}
}

func (m *ConfigEntry) SetValue(val interface{}) error {
	if !m.TryLock() {
		return ErrConfigEntryLockFailure
	}
	defer m.Unlock()

	m.Payload = val
	return nil
}

func (m *ConfigEntry) SetTarget(val interface{}) error {
	if !m.TryLock() {
		return ErrConfigEntryLockFailure
	}
	defer m.Unlock()

	m.Target = val
	return nil
}

func (m *ConfigEntry) Value() (val interface{}, e error) {
	if !m.TryRLock() {
		return nil, ErrConfigEntryLockFailure
	}
	defer m.RUnlock()

	if m.Step == -1 {
		val = m.Payload
		return
	}

	// smoothly logic
	return m.getPayload(rand.Intn(deployStep) + 1) // skipcq: GSC-G404 math/rand is enough
}

func (m *ConfigEntry) bootstrapDeploy() error {
	log.Trace().Msg("smooth deploy has been started")
	defer log.Trace().Msg("smooth deploy has been stopped")

	ticker, errors := time.NewTicker(deployInteration), 0

loop:
	for {
		select {
		case <-ticker.C:
			if ok, e := m.nextDeployStep(); e != nil {
				errors++
				log.Warn().Err(e).Msg("smooth_deploy - there some problems in runtime config deploying")

				if errors >= 3 {
					ticker.Stop()
					log.Error().Err(e).Msg("smooth-deploy has been crashed")
					return e
				}

				continue
			} else if ok {
				log.Debug().Msg("smooth_deploy - deploy prcess has been completed")
				break loop
			}
			log.Trace().Msg("smooth_deploy - tick called, descrease entry's steps")

		case <-done():
			log.Trace().Msg("smooth_deploy - internal abort() has been caught")
			break loop
		}
	}

	ticker.Stop()
	return m.commitTargetValue()
}

func (m *ConfigEntry) getLotteryResult(key int) (val interface{}) {
	switch key % m.Step {
	case 0:
		val = m.Target
	default:
		val = m.Payload
	}

	return
}

func (m *ConfigEntry) getPayload(bkey ...int) (val interface{}, e error) {
	if !m.TryRLock() {
		e = ErrConfigEntryLockFailure
		return
	}
	defer m.RUnlock()

	if m.Step == -1 {
		val = m.Payload
		return
	}

	bkey = append(bkey, 0) // default value
	val = m.getLotteryResult(bkey[0])

	return
}

func (m *ConfigEntry) nextDeployStep(init ...bool) (_ bool, e error) {
	init = append(init, false)

	if !m.TryLock() {
		e = ErrConfigEntryLockFailure
		return
	}
	defer m.Unlock()

	if init[0] {
		m.Step = deployStep
		return
	}

	log.Trace().Int("step", m.Step).Msg("")
	m.Step = m.Step - 1
	return m.Step == 0, e
}

func (m *ConfigEntry) commitTargetValue() (e error) {
	if !m.TryLock() {
		e = ErrConfigEntryLockFailure
		return
	}

	log.Debug().Interface("old", m.Payload).Interface("new", m.Target).
		Msg("config entry - new value has been commited")

	m.Payload, m.Step = m.Target, -1
	m.Unlock()
	return
}
