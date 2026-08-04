package runtime

import (
	"context"
	"errors"

	"github.com/MindHunter86/addie/utils"
	"github.com/rs/zerolog"
)

type RuntimePatchType uint8

const (
	RuntimePatchLottery RuntimePatchType = iota
	RuntimePatchQuality
	RuntimePatchAccessStdout
	RuntimePatchAccessLevel
	RuntimePatchQualityBypass
	RuntimePatchForceRUMitigate
)

var (
	ErrRuntimeUndefinedPatch = errors.New("given patch payload is undefined")

	RuntimeUtilsBindings = map[string]RuntimePatchType{
		utils.CfgLotteryChance:   RuntimePatchLottery,
		utils.CfgQualityLevel:    RuntimePatchQuality,
		utils.CfgAccessLogStdout: RuntimePatchAccessStdout,
		utils.CfgAccessLogLevel:  RuntimePatchAccessLevel,
		utils.CfgQualityBypass:   RuntimePatchQualityBypass,
		utils.CfgForceRUMitigate: RuntimePatchForceRUMitigate,
	}

	// intenal
	log *zerolog.Logger

	runtimeChangesHumanize = map[RuntimePatchType]string{
		RuntimePatchLottery:         "lottery chance",
		RuntimePatchQuality:         "quality level",
		RuntimePatchAccessStdout:    "access_log stdout switcher",
		RuntimePatchAccessLevel:     "access_log loglevel",
		RuntimePatchQualityBypass:   "quality rewrite bypass",
		RuntimePatchForceRUMitigate: "migrate unbypassed ru to europe",
	}
)

type (
	Runtime struct {
		Config *Storage
	}
)

func NewRuntime(c context.Context) (r *Runtime, e error) {
	log = c.Value(utils.ContextKeyLogger).(*zerolog.Logger)
	r = &Runtime{}

	if r.Config, e = NewStorage(c); e != nil {
		return
	}

	return
}
