package runtime

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/MindHunter86/addie/blocklist"
	"github.com/MindHunter86/addie/utils"
	"github.com/rs/zerolog"
)

type RuntimePatchType uint8

const (
	RuntimePatchLottery RuntimePatchType = iota
	RuntimePatchQuality
	RuntimePatchBlocklist
	RuntimePatchBlocklistIps
	RuntimePatchLimiter
	RuntimePatchA5bility
	RuntimePatchStdoutAccess
)

var (
	ErrRuntimeUndefinedPatch = errors.New("given patch payload is undefined")

	RuntimeUtilsBindings = map[string]RuntimePatchType{
		utils.CfgLotteryChance:     RuntimePatchLottery,
		utils.CfgQualityLevel:      RuntimePatchQuality,
		utils.CfgBlockList:         RuntimePatchBlocklistIps,
		utils.CfgBlockListSwitcher: RuntimePatchBlocklist,
		utils.CfgLimiterSwitcher:   RuntimePatchLimiter,
		utils.CfgClusterA5bility:   RuntimePatchA5bility,
		utils.CfgStdoutAccessLog:   RuntimePatchStdoutAccess,
	}

	// intenal
	log *zerolog.Logger

	runtimeChangesHumanize = map[RuntimePatchType]string{
		RuntimePatchLottery:      "lottery chance",
		RuntimePatchQuality:      "quality level",
		RuntimePatchBlocklist:    "blocklist switch",
		RuntimePatchBlocklistIps: "blocklist ips",
		RuntimePatchLimiter:      "limiter switch",
		RuntimePatchA5bility:     "balancer's cluster availability",
		RuntimePatchStdoutAccess: "stdout access log switcher",
	}
)

var smoothlyStats *RuntimeStats

type (
	Runtime struct {
		Config ConfigStorage
		Stats  *RuntimeStats

		// todo - refactor
		blocklist *blocklist.Blocklist // temporary;
	}
	RuntimePatch struct {
		Type  RuntimePatchType
		Patch []byte
	}

	RuntimeStats struct {
		sync.RWMutex
		sentPayload, sentTarget int
	}
)

func (m *RuntimeStats) SentPayload() {
	m.Lock()
	m.sentPayload++
	m.Unlock()
}

func (m *RuntimeStats) SentTarget() {
	m.Lock()
	m.sentTarget++
	m.Unlock()
}

func (m *RuntimeStats) Stats() (payload, target int) {
	m.RLock()
	defer m.RUnlock()

	payload, target = m.sentPayload, m.sentTarget
	return
}

func NewRuntime(c context.Context) (r *Runtime, e error) {
	blist := c.Value(utils.ContextKeyBlocklist).(*blocklist.Blocklist)
	log = c.Value(utils.ContextKeyLogger).(*zerolog.Logger)

	r = &Runtime{
		blocklist: blist,
		Stats:     &RuntimeStats{},
	}

	smoothlyStats = r.Stats

	if r.Config, e = NewConfigStorage(c); e != nil {
		return
	}

	return
}

func (m *Runtime) ApplyPatch(patch *RuntimePatch) (e error) {

	if len(patch.Patch) == 0 {
		return ErrRuntimeUndefinedPatch
	}

	switch patch.Type {
	case RuntimePatchLottery:
		e = patch.ApplyLotteryChance(&m.Config)

	case RuntimePatchQuality:
		e = patch.ApplyQualityLevel(&m.Config)
	case RuntimePatchBlocklistIps:
		e = patch.ApplyBlocklistIps(&m.Config, m.blocklist)

	case RuntimePatchBlocklist:
		e = patch.ApplySwitch(&m.Config, ConfigParamBlocklist)
	case RuntimePatchLimiter:
		e = patch.ApplySwitch(&m.Config, ConfigParamLimiter)

		// !!!
	case RuntimePatchA5bility:
		e = m.applyA5bility(patch.Patch)
	case RuntimePatchStdoutAccess:
		e = m.applyStdoutAccess(patch.Patch)

	default:
		panic("internal error - undefined runtime patch type")
	}

	if e != nil {
		log.Error().Err(e).
			Msgf("could not apply runtime configuration (%s)", runtimeChangesHumanize[patch.Type])
	}

	return
}

func (m *RuntimePatch) ApplyBlocklistIps(st *ConfigStorage, bl *blocklist.Blocklist) (e error) {
	buf := string(m.Patch)

	if buf == "_" {
		bl.Reset()
		log.Info().Msg("runtime patch has been for Blocklist.Reset")
		return
	}

	lastsize := bl.Size()
	ips := strings.Split(buf, ",")
	bl.Push(ips...)

	// dummy code
	// ???
	// st.SetValue(ConfigParamBlocklistIps, ips)

	log.Info().Msgf("runtime patch has been for Blocklist, applied %d ips", len(ips))
	log.Debug().Msgf("apply blocklist: last size - %d, new - %d", lastsize, bl.Size())
	return
}

func (m *RuntimePatch) ApplySwitch(st *ConfigStorage, param ConfigParam) (e error) {
	buf := string(m.Patch)

	switch buf {
	case "0":
		st.SetValue(param, 0)
	case "1":
		st.SetValue(param, 1)
	default:
		e = fmt.Errorf("invalid value in runtime bool patch for %d : %s", param, buf)
		return
	}

	log.Debug().Msgf("runtime patch has been applied for %d with %s", param, buf)
	return
}

func (m *RuntimePatch) ApplyQualityLevel(st *ConfigStorage) (e error) {
	buf := string(m.Patch)

	quality, ok := utils.GetTitleQualityByString[buf]
	if !ok {
		e = fmt.Errorf("quality is invalid; 480, 720, 1080 values are permited only, current - %s", buf)
		return
	}

	log.Info().Msgf("runtime patch has been applied for QualityLevel with %s", buf)
	st.SetValueSmoothly(ConfigParamQuality, quality)
	return
}

func (m *RuntimePatch) ApplyLotteryChance(st *ConfigStorage) (e error) {
	var chance int
	if chance, e = strconv.Atoi(string(m.Patch)); e != nil {
		return
	}

	if chance < 0 || chance > 100 {
		e = fmt.Errorf("chance could not be less than 0 and more than 100, current %d", chance)
		return
	}

	log.Info().Msgf("runtime patch has been applied for LotteryChance with %d", chance)
	st.SetValueSmoothly(ConfigParamLottery, chance)
	return
}

func (m *Runtime) StatsPrint() {
	for uid := range ConfigParamDefaults {
		name, ok := GetNameByConfigParam[uid]
		if !ok {
			continue
		}

		val, _, _ := m.Config.GetValue(uid)

		fmt.Printf("%s - %+v\n", name, val)
	}

	x, y := m.Stats.Stats()
	fmt.Printf("Payload - %d, Target - %d\n", x, y)
	// refval := reflect.ValueOf(m.Config)
	// reftype := reflect.TypeOf(m.Config)

	// for i := 0; i < refval.NumField(); i++ {
	// 	field := refval.Field(i)
	// 	fieldtype := reftype.Field(i)

	// }
}

// func (m *Runtime) Stats() io.Reader {
// 	tb := table.NewWriter()
// 	defer tb.Render()

// 	buf := bytes.NewBuffer(nil)
// 	tb.SetOutputMirror(buf)
// 	tb.AppendHeader(table.Row{
// 		"Key", "Value",
// 	})

// 	var runconfig = make(map[string]string)
// 	for key, bind := range RuntimeUtilsBindings {
// 		val, _, _ := m.Config.GetValue(bind).(string)
// 		runconfig[key] = val
// 	}

// 	servers := m.upstream.getServers(&m.ulock)

// 	for idx, server := range servers {
// 		var firstdiff, lastdiff float64

// 		if servers[0].handledRequests != 0 {
// 			firstdiff = (float64(server.handledRequests) * 100.00 / float64(servers[0].handledRequests)) - 100.00
// 		}

// 		if idx != 0 && servers[idx-1].handledRequests != 0 {
// 			lastdiff = (float64(server.handledRequests) * 100.00 / float64(servers[idx-1].handledRequests)) - 100.00
// 		}

// 		tb.AppendRow([]interface{}{
// 			server.Name, server.Ip,
// 			server.handledRequests, round(lastdiff, 2), round(firstdiff, 2), server.lastRequestTime.Format("2006-01-02T15:04:05.000"),
// 			isDownHumanize(server.isDown), server.lastChanged.Format("2006-01-02T15:04:05.000"),
// 		})
// 	}

// 	tb.Style().Options.SeparateRows = true

//		return buf
//	}
func (m *Runtime) applyA5bility(input []byte) (e error) {
	var percent int
	if percent, e = strconv.Atoi(string(input)); e != nil {
		return
	}

	if percent < 0 || percent > 100 {
		log.Warn().Int("percent", percent).Msg("percent could not be less than 0 and more than 100")
		return
	}

	log.Info().Msgf("runtime config - applied cluster availability %s", string(input))

	m.updateA5bility(percent)
	return
}

func (m *Runtime) applyStdoutAccess(input []byte) (e error) {
	var enabled int
	if enabled, e = strconv.Atoi(string(input)); e != nil {
		return
	}

	log.Trace().Msgf("runtime config - limiter apply value %d", enabled)

	switch enabled {
	case 0:
		fallthrough
	case 1:
		m.updateStdoutAccess(enabled)

		log.Info().Msg("runtime config - stdout-accesslog status updated")
		log.Debug().Msgf("apply stdout-accesslog - updated value - %d", enabled)
	default:
		log.Warn().Int("enabled", enabled).
			Msg("runtime config - stdout-accesslog could not be non 0 or non 1")
		return
	}
	return
}
