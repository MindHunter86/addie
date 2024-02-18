package runtime

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
)

var (
	ErrRuntimeUndefinedPatch = errors.New("given patch payload is undefined")

	RuntimeUtilsBindings = map[string]RuntimePatchType{
		utils.CfgLotteryChance:     RuntimePatchLottery,
		utils.CfgQualityLevel:      RuntimePatchQuality,
		utils.CfgBlockList:         RuntimePatchBlocklistIps,
		utils.CfgBlockListSwitcher: RuntimePatchBlocklist,
		utils.CfgLimiterSwitcher:   RuntimePatchLimiter,
	}

	// intenal
	log *zerolog.Logger

	runtimeChangesHumanize = map[RuntimePatchType]string{
		RuntimePatchLottery:      "lottery chance",
		RuntimePatchQuality:      "quality level",
		RuntimePatchBlocklist:    "blocklist switch",
		RuntimePatchBlocklistIps: "blocklist ips",
		RuntimePatchLimiter:      "limiter switch",
	}
)

type (
	Runtime struct {
		Config ConfigStorage

		// todo - refactor
		blocklist *blocklist.Blocklist // temporary;
	}
	RuntimePatch struct {
		Type  RuntimePatchType
		Patch []byte
	}
)

func NewRuntime(c context.Context) (r *Runtime, e error) {
	blist := c.Value(utils.ContextKeyBlocklist).(*blocklist.Blocklist)
	log = c.Value(utils.ContextKeyLogger).(*zerolog.Logger)

	r = &Runtime{
		blocklist: blist,
	}

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

func (m *RuntimePatch) ApplyValue(param ConfigParam, smoothly bool) (e error) {
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
	st.SetValue(ConfigParamQuality, quality)
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

func (m *Runtime) Stats() {
	for uid := range ConfigParamDefaults {
		name := GetNameByConfigParam[uid]
		val, _, _ := m.Config.GetValue(uid)

		fmt.Printf("%s - %+v\n", name, val)
	}
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

// 	return buf
// }
