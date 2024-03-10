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
	RuntimePatchAccessStdout
	RuntimePatchAccessLevel
)

var (
	ErrRuntimeUndefinedPatch = errors.New("given patch payload is undefined")

	RuntimeUtilsBindings = map[string]RuntimePatchType{
		utils.CfgLotteryChance:     RuntimePatchLottery,
		utils.CfgQualityLevel:      RuntimePatchQuality,
		utils.CfgBlockList:         RuntimePatchBlocklistIps,
		utils.CfgBlockListSwitcher: RuntimePatchBlocklist,
		utils.CfgLimiterSwitcher:   RuntimePatchLimiter,
		utils.CfgAccessLogStdout:   RuntimePatchAccessStdout,
		utils.CfgAccessLogLevel:    RuntimePatchAccessLevel,
	}

	// intenal
	log *zerolog.Logger

	runtimeChangesHumanize = map[RuntimePatchType]string{
		RuntimePatchLottery:      "lottery chance",
		RuntimePatchQuality:      "quality level",
		RuntimePatchBlocklist:    "blocklist switch",
		RuntimePatchBlocklistIps: "blocklist ips",
		RuntimePatchLimiter:      "limiter switch",
		RuntimePatchAccessStdout: "access_log stdout switcher",
		RuntimePatchAccessLevel:  "access_log loglevel",
	}
)

type (
	Runtime struct {
		Config *Storage

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

	if r.Config, e = NewStorage(c); e != nil {
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
		e = patch.ApplyLotteryChance(m.Config)

	case RuntimePatchQuality:
		e = patch.ApplyQualityLevel(m.Config)
	case RuntimePatchBlocklistIps:
		e = patch.ApplyBlocklistIps(m.Config, m.blocklist)

	case RuntimePatchBlocklist:
		e = patch.ApplySwitch(m.Config, ParamBlocklist)
	case RuntimePatchLimiter:
		e = patch.ApplySwitch(m.Config, ParamLimiter)
	case RuntimePatchAccessStdout:
		e = patch.ApplySwitch(m.Config, ParamAccessStdout)

	case RuntimePatchAccessLevel:
		e = patch.ApplyLogLevel(m.Config, ParamAccessLevel)

	default:
		panic("internal error - undefined runtime patch type")
	}

	if e != nil {
		log.Error().Err(e).
			Msgf("could not apply runtime configuration (%s)", runtimeChangesHumanize[patch.Type])
	}

	return
}

func (m *RuntimePatch) ApplyLogLevel(st *Storage, param StorageParam) (e error) {
	buf, level := strings.TrimSpace(string(m.Patch)), zerolog.NoLevel

	switch buf {
	case "trace":
		level = zerolog.TraceLevel
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	default:
		e = fmt.Errorf("unknown level received from consul for %s - %s", GetNameByParam[param], buf)
		return
	}

	st.Set(ParamAccessLevel, level)
	log.Info().Msgf("runtime patch has been applied for %s with %s", GetNameByParam[param], buf)
	return
}

func (m *RuntimePatch) ApplyBlocklistIps(_ *Storage, bl *blocklist.Blocklist) (e error) {
	buf := string(m.Patch)

	// TODO
	// !!! - fix unused code
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
	// st.SetValue(ParamBlocklistIps, ips)

	log.Info().Msgf("runtime patch has been for Blocklist, applied %d ips", len(ips))
	log.Debug().Msgf("apply blocklist: last size - %d, new - %d", lastsize, bl.Size())
	return
}

func (m *RuntimePatch) ApplySwitch(st *Storage, param StorageParam) (e error) {
	buf := string(m.Patch)

	switch buf {
	case "0":
		st.Set(param, 0)
	case "1":
		st.Set(param, 1)
	default:
		e = fmt.Errorf("invalid value in runtime bool patch for %s : %s", GetNameByParam[param], buf)
		return
	}

	log.Info().Msgf("runtime patch has been applied for %s with %s", GetNameByParam[param], buf)
	return
}

func (m *RuntimePatch) ApplyQualityLevel(st *Storage) (e error) {
	buf := string(m.Patch)

	quality, ok := utils.GetTitleQualityByString[buf]
	if !ok {
		e = fmt.Errorf("quality is invalid; 480, 720, 1080 values are permited only, current - %s", buf)
		return
	}

	log.Info().Msgf("runtime patch has been applied for QualityLevel with %s", buf)
	st.SetSmoothly(ParamQuality, quality)
	return
}

func (m *RuntimePatch) ApplyLotteryChance(st *Storage) (e error) {
	var chance int
	if chance, e = strconv.Atoi(string(m.Patch)); e != nil {
		return
	}

	if chance < 0 || chance > 100 {
		e = fmt.Errorf("chance could not be less than 0 and more than 100, current %d", chance)
		return
	}

	log.Info().Msgf("runtime patch has been applied for LotteryChance with %d", chance)
	st.SetSmoothly(ParamLottery, chance)
	return
}
