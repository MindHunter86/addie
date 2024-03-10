package utils

type ContextKey uint8

const (
	ContextKeyLogger ContextKey = iota
	ContextKeyCliContext
	ContextKeyAbortFunc
	ContextKeyRPatcher
	ContextKeyBlocklist
	ContextKeyRuntime
	ContextKeyBalancers
)

const (
	FbReqTmruestTimer ContextKey = iota
)

const (
	FbReqTmrPreCond ContextKey = iota
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
	CfgStdoutAccessLog   = "stdout-access-log"
	CfgAccessLogStdout   = "access-log-stdout"
	CfgAccessLogLevel    = "access-log-level"
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

func JoinSize(size int, s ...[]byte) []byte {
	b, i := make([]byte, size), 0
	for _, v := range s {
		i += copy(b[i:], v)
	}
	return b
}
