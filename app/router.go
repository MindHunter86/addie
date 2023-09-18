package app

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MindHunter86/anilibria-hlp-service/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/skip"
	"github.com/rs/zerolog"
)

func (m *App) fiberConfigure() {
	// time collector + logger
	m.fb.Use(func(c *fiber.Ctx) (e error) {

		c.SetUserContext(context.WithValue(
			c.UserContext(),
			utils.FbReqTmruestTimer,
			make(map[utils.ContextKey]time.Time),
		))

		start, e := time.Now(), c.Next()
		stop := time.Now()

		if !strings.HasPrefix(c.Path(), "/videos/media/ts") {
			gLog.Trace().Msg("non sign request detected, skipping timings...")
			return
		}

		total := stop.Sub(start).Round(time.Microsecond)
		setup := stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrBeforeRoute)).Round(time.Microsecond)
		routing := stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrPreCond)).Round(time.Microsecond)
		precond := stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrBlocklist)).Round(time.Microsecond)
		blist := stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrFakeQuality)).Round(time.Microsecond)
		fquality := stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrConsulLottery)).Round(time.Microsecond)
		clottery := stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrReqSign)).Round(time.Microsecond)
		reqsign := stop.Sub(stop).Round(time.Microsecond)

		reqsign = clottery - reqsign
		clottery = fquality - clottery
		fquality = blist - fquality
		blist = precond - blist
		precond = routing - precond
		routing = setup - routing
		setup = total - setup

		if gLog.GetLevel() <= zerolog.DebugLevel {
			gLog.Debug().
				Str("id", c.Locals("requestid").(string)).
				Dur("setup", setup).
				Dur("routing", routing).
				Dur("precond", precond).
				Dur("blist", blist).
				Dur("fquality", fquality).
				Dur("clottery", clottery).
				Dur("reqsign", reqsign).
				Dur("total", total).
				Dur("timer", time.Since(stop).Round(time.Microsecond)).
				Msg("")

			gLog.Trace().Msgf(
				"Total: %s, Setup %s; Routing %s; PreCond %s; Blocklist %s; FQuality %s; CLottery %s; ReqSign %s;",
				total, setup, routing, precond, blist, fquality, clottery, reqsign)
			gLog.Trace().Msgf("Time Collector %s", time.Since(stop).Round(time.Microsecond))
		}

		if gLog.GetLevel() <= zerolog.InfoLevel {
			gLog.Info().
				Int("status", c.Response().StatusCode()).
				Str("method", c.Method()).
				Str("path", c.Path()).
				Str("id", c.Locals("requestid").(string)).
				Str("ip", c.IP()).
				Dur("latency", total).
				Str("user-agent", c.Get(fiber.HeaderUserAgent)).
				Msg("")
		}

		return
	})

	// panic recover for all handlers
	m.fb.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			gLog.Error().Str("request", c.Request().String()).Bytes("stack", debug.Stack()).
				Msg("panic has been caught")
			_, _ = os.Stderr.WriteString(fmt.Sprintf("panic: %v\n%s\n", e, debug.Stack())) //nolint:errcheck // This will never fail
		},
	}))

	// debug
	if gCli.Bool("http-pprof-enable") {
		m.fb.Use(pprof.New())
	}

	// request id
	m.fb.Use(requestid.New())

	// favicon disable
	m.fb.Use(favicon.New(favicon.ConfigDefault))

	// compress support
	m.fb.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// CORS serving
	if gCli.Bool("http-cors") {
		m.fb.Use(cors.New(cors.Config{
			AllowOrigins: "*",
			AllowHeaders: strings.Join([]string{
				fiber.HeaderContentType,
			}, ","),
			AllowMethods: strings.Join([]string{
				fiber.MethodPost,
			}, ","),
		}))
	}

	// time collector - Before routing
	m.fb.Use(func(c *fiber.Ctx) error {
		m.lapRequestTimer(c, utils.FbReqTmrBeforeRoute)
		return c.Next()
	})

	// API Basic Auth config
	creds := gCli.String("api-credentials")
	baConfig := basicauth.Config{
		Users: map[string]string{
			strings.Split(creds, ":")[0]: strings.Split(creds, ":")[1],
		},
	}

	// Routes

	m.fb.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// group api - /api
	api := m.fb.Group("/api", basicauth.New(baConfig))
	api.Post("logger/level", m.fbHndApiLoggerLevel)
	api.Post("limiter/switch", m.fbHndApiLimiterSwitch)

	// group upstream
	upstr := api.Group("/balancer")
	upstr.Get("/stats", m.fbHndApiBalancerStats)
	upstr.Post("/stats/reset", m.fbHndApiStatsReset)
	upstr.Post("/reset", m.fbHndApiBalancerReset)

	upstrCluster := upstr.Group("/cluster", skip.New(m.fbHndApiPreCondErr, m.fbMidBlcPreCond))
	upstrCluster.Get("/cache-nodes",
		m.fbHndBlcNodesBalance,
		m.fbHndBlcNodesBalanceFallback)

	// group blocklist - /api/blocklist
	blist := api.Group("/blocklist")
	blist.Post("/add", m.fbHndApiBlockIp)
	blist.Post("/remove", m.fbHndApiUnblockIp)
	blist.Post("/switch", m.fbHndApiBListSwitch)
	blist.Post("/reset", m.fbHndApiBlockReset)

	// group media - /videos/media/ts
	media := m.fb.Group("/videos/media/ts", skip.New(m.fbHndApiPreCondErr, m.fbMidAppPreCond))

	// group media - blocklist
	media.Use(m.fbMidAppBlocklist)

	// group media - limiter
	media.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			if limiting, ok := m.runtime.GetLimiterStatus(); limiting == 0 || !ok {
				return true
			}

			return c.IP() == "127.0.0.1" || gCli.App.Version == "devel"
		},

		Max:        gCli.Int("limiter-max-req"),
		Expiration: gCli.Duration("limiter-records-duration"),

		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},

		Storage: m.fbstor,
	}))

	// group media - middlewares
	media.Use(m.fbMidAppFakeQuality)
	media.Use(skip.New(m.fbMidAppBalance, m.fbMidAppBalancerLottery))

	// group media - sign handler
	media.Use(m.fbHndAppRequestSign)
}

func (*App) lapRequestTimer(c *fiber.Ctx, k utils.ContextKey) {
	c.UserContext().
		Value(utils.FbReqTmruestTimer).(map[utils.ContextKey]time.Time)[k] = time.Now()
}

func (*App) getRequestTimerSegment(c *fiber.Ctx, k utils.ContextKey) time.Time {
	return c.UserContext().
		Value(utils.FbReqTmruestTimer).(map[utils.ContextKey]time.Time)[k]
}
