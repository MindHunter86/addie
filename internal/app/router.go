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

	"github.com/MindHunter86/addie/internal/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/skip"
	"github.com/rs/zerolog"
)

func (m *App) HandleRoutes(fapp *fiber.App) {

	// panic recover for all handlers
	fapp.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			rlog(c).Error().Str("request", c.Request().String()).Bytes("stack", debug.Stack()).
				Msg("panic has been caught")
			_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", e, debug.Stack()) //nolint:errcheck // This will never fail

			c.Status(fiber.StatusInternalServerError)
		},
	}))

	// request id
	fapp.Use(requestid.New())

	// prefixed logger initialization
	// - we send logs in syslog and stdout by default,
	// - but if access-log-stdout is 0 we use syslog output only
	fapp.Use(func(c *fiber.Ctx) error {
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
	fapp.Use(func(c *fiber.Ctx) (e error) {
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
			routing, precond, fquality, clottery, reqsign :=
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrPreCond)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrFakeQuality)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrConsulLottery)).Round(time.Microsecond),
				stop.Sub(m.getRequestTimerSegment(c, utils.FbReqTmrReqSign)).Round(time.Microsecond),
				stop.Sub(stop).Round(time.Microsecond)

			reqsign = clottery - reqsign
			clottery = fquality - clottery
			fquality = precond - fquality
			precond = routing - precond
			routing = total - routing

			rlog(c).Debug().
				Dur("routing", routing).
				Dur("precond", precond).
				Dur("fquality", fquality).
				Dur("clottery", clottery).
				Dur("reqsign", reqsign).
				Dur("total", total).
				Dur("timer", time.Since(stop).Round(time.Microsecond)).
				Msg("")

			rlog(c).Trace().Msgf(
				"Total: %s; Routing %s; PreCond %s; FQuality %s; CLottery %s; ReqSign %s;",
				total, routing, precond, fquality, clottery, reqsign)
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
		fapp.Use(pprof.New())
	}

	// favicon disable
	fapp.Use(favicon.New(favicon.ConfigDefault))

	// compress support
	fapp.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// CORS serving
	if gCli.Bool("http-cors") {
		fapp.Use(cors.New(cors.Config{
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

	// group api - /api
	api := fapp.Group("/api")

	// TODO - waiting migration on Dynamic Config
	// api.Post("logger/level", gController.SetLoggerLevel)
	// api.Post("quality", gController.UpdateQualityRewrite)

	// group upstream
	upstr := api.Group("/balancer")
	upstr.Get("/stats", gController.GetBalancerStats)
	upstr.Post("/stats/reset", gController.BalancerStatsReset)

	upstrCluster := upstr.Group("/cluster", skip.New(m.fbHndApiPreCondErr, m.fbMidBlcPreCond))
	upstrCluster.Get("/cache-nodes",
		m.fbHndBlcNodesBalance,
		m.fbHndBlcNodesBalanceFallback)

	// group media - /videos/media/ts
	media := fapp.Group("/videos/media/ts", skip.New(m.fbHndApiPreCondErr, m.fbMidAppPreCond))

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
