package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MindHunter86/addie/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/skip"
	"github.com/rs/zerolog"

	_ "github.com/MindHunter86/addie/docs"
	"github.com/gofiber/swagger"
)

// @title Fiber Example API
// @version 1.0
// @description This is a sample swagger for Fiber
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /
func (m *App) fiberConfigure() {

	// panic recover for all handlers
	m.fb.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			rlog(c).Error().Str("request", c.Request().String()).Bytes("stack", debug.Stack()).
				Msg("panic has been caught")
			_, _ = os.Stderr.WriteString(fmt.Sprintf("panic: %v\n%s\n", e, debug.Stack())) //nolint:errcheck // This will never fail

			c.Status(fiber.StatusInternalServerError)
		},
	}))

	// request id
	m.fb.Use(requestid.New())

	// prefixed logger initialization
	// - we send logs in syslog and stdout by default,
	// - but if access-log-stdout is 0 we use syslog output only
	m.fb.Use(func(c *fiber.Ctx) error {
		logger := gLog.With().Str("id", c.Locals("requestid").(string)).Logger().
			Level(m.runtime.Config.Get(runtime.ParamAccessLevel).(zerolog.Level))
		syslogger := logger.Output(m.syslogWriter)

		if m.runtime.Config.Get(runtime.ParamAccessStdout).(int) == 0 {
			logger = logger.Output(io.Discard)
		}

		c.Locals("logger", &logger)
		c.Locals("syslogger", &syslogger)
		return c.Next()
	})

	// time collector + logger
	m.fb.Use(func(c *fiber.Ctx) (e error) {
		if !strings.HasPrefix(c.Path(), "/videos/media/ts") &&
			!strings.HasPrefix(c.Path(), "/api/balancer/cluster") {
			// rlog(c).Trace().Str("path", c.Path()).Msg("non sign request detected, skipping timings...")
			return c.Next()
		}

		c.SetUserContext(context.WithValue(
			c.UserContext(),
			utils.FbReqTmruestTimer,
			make(map[utils.ContextKey]time.Time),
		))

		start, e := time.Now(), c.Next()
		stop := time.Now()
		total := stop.Sub(start).Round(time.Microsecond)

		status, lvl, err := c.Response().StatusCode(), zerolog.InfoLevel, new(fiber.Error)
		if errors.As(e, &err) || status >= fiber.StatusInternalServerError {
			status, lvl = err.Code, zerolog.WarnLevel
		}

		if rlog(c).GetLevel() <= zerolog.DebugLevel {
			routing, precond, blist, fquality, clottery, reqsign :=
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrPreCond)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrBlocklist)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrFakeQuality)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrConsulLottery)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrReqSign)).Round(time.Microsecond),
				stop.Sub(stop).Round(time.Microsecond)

			reqsign = clottery - reqsign
			clottery = fquality - clottery
			fquality = blist - fquality
			blist = precond - blist
			precond = routing - precond
			routing = total - routing

			rlog(c).Debug().
				Dur("routing", routing).
				Dur("precond", precond).
				Dur("blist", blist).
				Dur("fquality", fquality).
				Dur("clottery", clottery).
				Dur("reqsign", reqsign).
				Dur("total", total).
				Dur("timer", time.Since(stop).Round(time.Microsecond)).
				Msg("")

			rlog(c).Trace().Msgf(
				"Total: %s; Routing %s; PreCond %s; Blocklist %s; FQuality %s; CLottery %s; ReqSign %s;",
				total, routing, precond, blist, fquality, clottery, reqsign)
			rlog(c).Trace().Msgf("Time Collector %s", time.Since(stop).Round(time.Microsecond))
		}

		rlog(c).WithLevel(lvl).
			Int("status", status).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Dur("latency", total).
			Str("user-agent", c.Get(fiber.HeaderUserAgent)).Msg("")
		m.rsyslog(c).WithLevel(lvl).
			Int("status", status).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Dur("latency", total).
			Str("user-agent", c.Get(fiber.HeaderUserAgent)).Msg("")

		return
	})

	// debug
	if gCli.Bool("http-pprof-enable") {
		m.fb.Use(pprof.New())
	}

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

	// Routes

	// swagger
	m.fb.Get("/swagger/*", swagger.HandlerDefault)

	// group api - /api
	api := m.fb.Group("/api")
	api.Post("logger/level", gController.SetLoggerLevel)
	api.Post("limiter/switch", gController.LimiterSwitch)
	api.Post("quality", gController.UpdateQualityRewrite)

	// group upstream
	upstr := api.Group("/balancer")
	upstr.Get("/stats", gController.GetBalancerStats)
	upstr.Post("/stats/reset", gController.BalancerStatsReset)
	upstr.Post("/reset", gController.BalancerUpstreamReset)

	upstrCluster := upstr.Group("/cluster", skip.New(m.fbHndApiPreCondErr, m.fbMidBlcPreCond))
	upstrCluster.Get("/cache-nodes",
		m.fbHndBlcNodesBalance,
		m.fbHndBlcNodesBalanceFallback)

	// group blocklist - /api/blocklist
	blist := api.Group("/blocklist")
	blist.Post("/add", gController.BlockIP)
	blist.Post("/remove", gController.UnblockIP)
	blist.Post("/switch", gController.BlocklistSwitch)
	blist.Post("/reset", gController.BlocklistReset)

	// group media - /videos/media/ts
	media := m.fb.Group("/videos/media/ts", skip.New(m.fbHndApiPreCondErr, m.fbMidAppPreCond))

	// group media - blocklist & limiter
	media.Use(m.fbMidAppBlocklist)
	media.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			if m.runtime.Config.Get(runtime.ParamLimiter).(int) == 0 {
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

	// group media - core cache fetcher
	media.Use(m.fbHndApiCoreBalance)

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
