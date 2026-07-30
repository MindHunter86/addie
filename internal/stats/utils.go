package stats

import (
	"sync"
)

type IncrementMetric uint16
type AverageMetric uint16
type PersistentMetric uint16

const (
	// increment metrics
	IMHTTPServerRequest IncrementMetric = iota
	IMHTTPServerInvalidRequest
	IMHTTPServerPanic
	IMHTTPServerCode100
	IMHTTPServerCode200
	IMHTTPServerCode300
	IMHTTPServerCode400
	IMHTTPServerCode500
	IMHTTPServerNoCode

	IMFiberReqPathStatic
	IMFiberReqPathAssets
	IMFiberReqPathAuthMod
	IMFiberReqPathVerify
	IMFiberReqPathRoot
	IMFiberReqPathSettings
	IMFiberReqPathConfigPatch

	IMDBCacheHit
	IMDBCacheMiss
	IMDBCacheUpdate
	IMDBTxGet
	IMDBTxUpdate

	IMAppVldReqBypass
	IMAppVldReqForceCheck
	IMAppVldReqNonLockdown

	IMAppVldErrReqRequires
	IMAppVldErrNLength
	IMAppVldErrAbnrmlModSize
	IMAppVldErrSLength
	IMAppVldErrLengthMismatch

	IMAppAuthErrReqRequires
	IMAppAuthErrRawDataParse
	IMAppAuthErrNoPayloadInDB
	IMAppAuthErrPayloadExpired
	IMAppAuthSignKeysizeUndef
	IMAppAuthSignKeysize2048
	IMAppAuthSignKeysize4096
	IMAppAuthSignKeysize8192
	IMAppAuthRequestOK

	IMAppVfyErrInvalidModulus
	IMAppVfyErrBase64
	IMAppVfyErrModulusVerify
	IMAppVfyErrRawDataParse
	IMAppVfyErrPayloadExpired
	IMAppVfyErrModulSizeDif
	IMAppVfyErrSignVerify
	IMAppVfyErrDatabaseUpdate
	IMAppVfyRequestOK

	IMStatsLoopCount
	IMStatsLoopLockedError

	IMAnltLoopCount
	IMAnltLoopLockedError
	IMAnltLockdownEnable
	IMAnltLockdownDisable
	IMAnltDBRequests
	IMAnltDBReqErrors

	IMSvcKernSignDebug

	// average metrics
	AMHTTPServerLatency AverageMetric = iota

	AMAnltStatReqCurr
	AMAnltStatReqDiff
	AMAnltStatReqRatio
	AMAnltDBReqTime

	// persistent metrics
	// PMSvcModulesRunning PersistentMetric = iota

	_metricsEOL
)

var MetricNames = map[any]string{
	IMHTTPServerRequest:        "fiber.requests",
	IMHTTPServerInvalidRequest: "fiber.requests_invalid",
	IMHTTPServerPanic:          "fiber.requests_panics",
	IMHTTPServerCode100:        "fiber.requests_100",
	IMHTTPServerCode200:        "fiber.requests_200",
	IMHTTPServerCode300:        "fiber.requests_300",
	IMHTTPServerCode400:        "fiber.requests_400",
	IMHTTPServerCode500:        "fiber.requests_500",
	IMHTTPServerNoCode:         "fiber.requests_000",
	IMDBCacheHit:               "db.cache_hit",
	IMDBCacheMiss:              "db.cache_miss",
	IMDBCacheUpdate:            "db.cache_update",
	IMDBTxGet:                  "db.tx_get",
	IMDBTxUpdate:               "db.tx_update",
	IMFiberReqPathStatic:       "fiber.path.static",
	IMFiberReqPathAssets:       "fiber.path.assets",
	IMFiberReqPathAuthMod:      "fiber.path.authmod",
	IMFiberReqPathVerify:       "fiber.path.verify",
	IMFiberReqPathRoot:         "fiber.path.root",
	IMFiberReqPathSettings:     "fiber.path.settings",
	IMFiberReqPathConfigPatch:  "fiber.path.config_patch",
	IMAppVldReqBypass:          "app.vld_reqbypass",
	IMAppVldReqForceCheck:      "app.vld_reqforcecheck",
	IMAppVldReqNonLockdown:     "app.vld_reqnonlockdown",
	IMAppVldErrReqRequires:     "app.vlderror_reqrequirements",
	IMAppVldErrNLength:         "app.vlderror_nlength",
	IMAppVldErrAbnrmlModSize:   "app.vlderror_abnrmlmodsize",
	IMAppVldErrSLength:         "app.vlderror_slength",
	IMAppVldErrLengthMismatch:  "app.vlderror_lengthmismatch",
	IMAppAuthErrReqRequires:    "app.autherror_reqrequirements",
	IMAppAuthErrRawDataParse:   "app.autherror_rawdataparse",
	IMAppAuthErrNoPayloadInDB:  "app.autherror_nopayloadindb",
	IMAppAuthErrPayloadExpired: "app.autherror_payloadexpire",
	IMAppAuthSignKeysizeUndef:  "app.sign_keysizeundef",
	IMAppAuthSignKeysize2048:   "app.sign_keysize2048",
	IMAppAuthSignKeysize4096:   "app.sign_keysize4096",
	IMAppAuthSignKeysize8192:   "app.sign_keysize8192",
	IMAppAuthRequestOK:         "app.auth_requestok",
	IMAppVfyErrInvalidModulus:  "app.vfyerror_invalidmodulus",
	IMAppVfyErrBase64:          "app.vfyerror_base64",
	IMAppVfyErrModulusVerify:   "app.vfyerror_modulusverify",
	IMAppVfyErrRawDataParse:    "app.vfyerror_rawdataparse",
	IMAppVfyErrPayloadExpired:  "app.vfyerror_payloadexpire",
	IMAppVfyErrModulSizeDif:    "app.vfyerror_modulussizedif",
	IMAppVfyErrSignVerify:      "app.vfyerror_signverify",
	IMAppVfyErrDatabaseUpdate:  "app.vfyerror_databaseupdate",
	IMAppVfyRequestOK:          "app.verify_requestok",
	IMStatsLoopCount:           "stats.loop_count",
	IMStatsLoopLockedError:     "stats.loop_lockederr",
	IMAnltLoopCount:            "analytics.loop_count",
	IMAnltLoopLockedError:      "analytics.loop_lockederr",
	IMAnltLockdownEnable:       "analytics.lockdown_on",
	IMAnltLockdownDisable:      "analytics.lockdown_off",
	IMAnltDBRequests:           "analytics.db_requests",
	IMAnltDBReqErrors:          "analytics.db_reqerrs",
	IMSvcKernSignDebug:         "service.kernsign_debug_calls",

	AMHTTPServerLatency: "fiber.latency_ns",
	AMAnltStatReqCurr:   "analytics.statreq_current",
	AMAnltStatReqDiff:   "analytics.statreq_differ",
	AMAnltStatReqRatio:  "analytics.statreq_ratio",
	AMAnltDBReqTime:     "analytics.db_latency_ns",
}

var queueTaskPool = sync.Pool{New: func() any {
	return &queueTask{}
}}

func acquireQueueTask() *queueTask  { return queueTaskPool.Get().(*queueTask) }
func releaseQueueTask(v *queueTask) { queueTaskPool.Put(v) }
