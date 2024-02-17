package utils

type ContextKey uint8

const (
	ContextKeyLogger ContextKey = iota
	ContextKeyCliContext
	ContextKeyAbortFunc
	ContextKeyRPatcher
	ContextKeyBlocklist
	ContextKeyBalancers
)

const (
	FbReqTmruestTimer ContextKey = iota
)

const (
	FbReqTmrBeforeRoute ContextKey = iota
	FbReqTmrPreCond
	FbReqTmrBlocklist
	FbReqTmrFakeQuality
	FbReqTmrConsulLottery
	FbReqTmrReqSign
	FbReqTmrBlcPreCond
)

const (
	CfgLotteryChance     = "lottery-chance"
	CfgQualityLevel      = "quality-level"
	CfgBlockList         = "block-list"
	CfgBlockListSwitcher = "block-list-switcher"
	CfgLimiterSwitcher   = "limiter-switcher"
)

const (
	ChunkPath = iota + 1
	ChunkTitleId
	ChunkEpisodeId
	ChunkQualityLevel
	ChunkName
)

type TitleQuality uint8

const (
	TitleQualityNone TitleQuality = iota
	TitleQualitySD
	TitleQualityHD
	TitleQualityFHD
)

var GetTitleQualityByString = map[string]TitleQuality{
	"480":  TitleQualitySD,
	"720":  TitleQualityHD,
	"1080": TitleQualityFHD,
}

func (m *TitleQuality) String() string {
	switch *m {
	case TitleQualitySD:
		return "480"
	case TitleQualityHD:
		return "720"
	case TitleQualityFHD:
		return "1080"
	default:
		return ""
	}
}
