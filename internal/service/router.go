package service

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MindHunter86/addie/internal/stats"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"github.com/valyala/tcplisten"
)

func (m *Service) fiberMiddlewareInitialization() {
	// pprof profiler
	// manual:
	// 	curl -o profile.out https://host/debug/pprof -H 'X-Authorization: $TOKEN'
	// 	go tool pprof profile.out
	ppEnabled, ppSecret :=
		m.cli.Bool("http-pprof-enable"),
		utils.UnsafeBytes(m.cli.String("http-pprof-secret"))

	m.fb.Use(pprof.New(pprof.Config{
		Next: func(c *fiber.Ctx) bool {
			if !ppEnabled {
				return true
			}

			return !bytes.Equal(ppSecret, c.Request().Header.Peek("x-pprof-secret"))
		},
		Prefix: m.cli.String("http-pprof-prefix"),
	}))

	// Small middleware for optimizing working with IP
	// and reducing allocs by using net.IP package
	m.fb.Use(func(c *fiber.Ctx) (e error) {
		_, e = utils.IPFromFiberRequest(c), c.Next()

		c.Response().Header.Del(utils.IP_RESPONSE_HEADER)
		return e
	})

	// simple limiter for all requests
	limitMaxRps := m.cli.Int("limit-request-maxrps")
	m.fb.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return limitMaxRps == 0 || utils.IPFromFiberRequest(c) == "127.0.0.1"
		},
		KeyGenerator: utils.IPFromFiberRequest,

		Max:        m.cli.Int("limit-request-maxrps"),
		Expiration: m.cli.Duration("limit-request-expiration"),
	}))

	// push global app context for request
	m.fb.Use(func(c *fiber.Ctx) (e error) {
		c.SetUserContext(m.ctx)
		return c.Next()
	})

	// panic recover for all handlers
	m.fb.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			utils.Rlog(c, zerolog.ErrorLevel).Str("request", c.Request().String()).Bytes("stack", debug.Stack()).
				Msg("panic has been caught")
			_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", e, debug.Stack()) //nolint:errcheck // This will never fail

			c.Status(fiber.StatusInternalServerError)

			// stat paniced request
			ctx := c.UserContext()
			sts := utils.ContextValueExtract[*stats.Stats](ctx, utils.CtxStats)

			sts.WriteIncMetric(stats.IMHTTPServerPanic)
		},
	}))

	// request id generation with saving in context
	m.fb.Use(requestid.New())

	// !!! TODO - TEST AND DROP
	// prefixed logger initialization
	// - we send logs in syslog and stdout by default,
	// - but if access-log-stdout is 0 we use syslog output only
	// m.fb.Use(func(c *fiber.Ctx) error {
	// 	logger := m.log.With().Str("id", c.Locals("requestid").(string)).Logger().
	// 		Level(m.runtime.Config.Get(runtime.ParamAccessLevel).(zerolog.Level))
	// 	syslogger := logger.Output(m.syslogWriter)

	// 	if m.runtime.Config.Get(runtime.ParamAccessStdout).(int) == 0 {
	// 		logger = logger.Output(io.Discard)
	// 	}

	// 	c.Locals("logger", &logger)
	// 	c.Locals("syslogger", &syslogger)
	// 	return c.Next()
	// })

	// time collector + logger + stats
	m.fb.Use(func(c *fiber.Ctx) (e error) {
		started, e := time.Now(), c.Next()

		status, lvl := c.Response().StatusCode(), utils.HTTPAccessLogLevel

		// defaults for errored requests
		var cause string
		if e != nil {
			cause, lvl = e.Error(), zerolog.ErrorLevel
			status = fiber.StatusInternalServerError
		}

		// redefine variables for Fiber errors
		switch e := e.(type) { // skipcq: CRT-A0014 type should be inside switch
		case *fiber.Error:
			if e.Code > fiber.StatusBadRequest && e.Code < fiber.StatusInternalServerError {
				cause, lvl = e.Error(), zerolog.WarnLevel
			} else if e.Code > fiber.StatusInternalServerError {
				cause, lvl = e.Error(), zerolog.ErrorLevel
			}

			status = e.Code
		}

		// dump request data for debugging "error" cases
		if lvl >= zerolog.ErrorLevel && cause != "" {
			utils.Rlog(c, lvl).Msg(c.Request().String())
		}

		// utils.Rlog(c, lvl).
		// 	Int("status", status).
		// 	Str("method", c.Method()).
		// 	Str("path", c.Path()).
		// 	Str("ip", utils.IPFromFiberRequest(c)).
		// 	Dur("latency", elapsed).
		// 	Str("user-agent", c.Get(fiber.HeaderUserAgent)).Msg(cause)

		// todo : need some tests in production, revert if causes errs
		elapsed := time.Since(started)
		utils.RlogFast(c, lvl, status, elapsed, cause)

		// stats record
		ctx := c.UserContext()
		sts := utils.ContextValueExtract[*stats.Stats](ctx, utils.CtxStats)

		if sts != nil {
			sts.WriteIncMetric(stats.IMHTTPServerRequest)
			sts.WriteAvgMetric(stats.AMHTTPServerLatency, uint64(elapsed.Nanoseconds()))

			// sts.QueueWriteMetric(stats.IMHTTPServerRequest)
			if status >= 100 && status <= 199 {
				sts.WriteIncMetric(stats.IMHTTPServerCode100)
			} else if status >= 200 && status <= 299 {
				// sts.QueueWriteMetric(stats.IMHTTPServerCode200)
				sts.WriteIncMetric(stats.IMHTTPServerCode200)
			} else if status >= 300 && status <= 399 {
				// sts.QueueWriteMetric(stats.IMHTTPServerCode300)
				sts.WriteIncMetric(stats.IMHTTPServerCode300)
			} else if status >= 400 && status <= 499 {
				// sts.QueueWriteMetric(stats.IMHTTPServerCode400)
				sts.WriteIncMetric(stats.IMHTTPServerCode400)
			} else if status >= 500 && status <= 599 {
				// sts.QueueWriteMetric(stats.IMHTTPServerCode500)
				sts.WriteIncMetric(stats.IMHTTPServerCode500)
			} else {
				sts.WriteIncMetric(stats.IMHTTPServerNoCode)
			}
		}

		return
	})

	// favicon disabler
	m.fb.Use(favicon.New(favicon.ConfigDefault))

	// compression support
	m.fb.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// !! TODO - reviewme
	// CORS serving
	if m.cli.Bool("http-cors") {
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
}

func (m *Service) fiberRouterInitialization() {
	//
	//	Router pre-initialization
	//

	// dynamic settings and helpers:
	statsToken := utils.UnsafeBytes(m.cli.String("http-stats-secret"))

	// TODO : resolve with app.HandleRouter
	// controller :=
	// 	utils.ContextValueExtract[*app.Controller](m.ctx, utils.CtxRuntime)

	// basic auth for settings page
	// settingsPageBAuth := basicauth.New(basicauth.Config{
	// 	Users: map[string]string{
	// 		m.cli.String("internals-auth-username"): m.cli.String("internals-auth-password"),
	// 	},

	// 	Realm: "Addie Internals",
	// })

	//
	//	Router handlers configuration
	//

	// routing base
	apiv1 := m.fb.Group("/.within.website/x/cmd/" + m.fb.Config().AppName)

	// internal settings page
	internal := apiv1.Group("/internal")
	internal.Options("/config", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "PATCH")
		c.Set("Access-Control-Allow-Headers", "cache-control,x-requested-with")
		return c.SendStatus(204)
	})
	// internal.Patch("/api/config",
	// 	statIncMetricFiberHandler(stats.IMFiberReqPathConfigPatch), settingsPageBAuth,
	// 	dynamic.ConfigPatchHandler)

	// expvars stats page
	internal.Get("/stats", func(c *fiber.Ctx) error {
		if !bytes.Equal(c.Request().Header.Peek(fasthttp.HeaderAuthorization), statsToken) {
			c.Status(fiber.StatusNotFound)
			return fiber404ErrorHandler(c)
		}

		expvarhandler.ExpvarHandler(c.Context())
		return nil
	})

	// challenge methods
	// !! LOOK UP FOR STATS USAGE
	// !! LOOK UP FOR STATS USAGE
	// !! LOOK UP FOR STATS USAGE
	// ichallenge := apiv1.Group("/identity-challenge", statIncMetricFiberHandler(stats.IMFiberReqPathVerify),
	// 	challenge.FiberIChallengePreValidation(chCfg))
	// ichallenge.Get("/verify/2048", challenge.FiberIChallengeValidation(chCfg, 2048))
	// ichallenge.Get("/verify/4096", challenge.FiberIChallengeValidation(chCfg, 4096))
	// ichallenge.Get("/verify/8192", challenge.FiberIChallengeValidation(chCfg, 8192))

	// // nginx auth_request methods
	// nginx := apiv1.Group("/nginx-auth", statIncMetricFiberHandler(stats.IMFiberReqPathAuthMod),
	// 	challenge.FiberRequestControlHeaders(chCfg))
	// nginx.Get("/authorize", challenge.FiberAuthRequestPreValidation(chCfg))

	// Routes

	// !!! TODO
	// ! Add support for encrypted urls
	// m.fb.Get("/payload/:payload", encrypted)

	// group api - /api
	// api := m.fb.Group("/api")

	// TODO - waiting migration on Dynamic Config
	// api.Post("logger/level", controller.SetLoggerLevel)
	// api.Post("quality", controller.UpdateQualityRewrite)

	// group upstream
	// TODO : resolve with app.HandleRouter
	// upstr := api.Group("/balancer")
	// upstr.Get("/stats", controller.GetBalancerStats)
	// upstr.Post("/stats/reset", controller.BalancerStatsReset)

	// upstrCluster := upstr.Group("/cluster", skip.New(m.fbHndApiPreCondErr, m.fbMidBlcPreCond))
	// upstrCluster.Get("/cache-nodes",
	// 	m.fbHndBlcNodesBalance,
	// 	m.fbHndBlcNodesBalanceFallback)

	// // group media - /videos/media/ts
	// media := m.fb.Group("/videos/media/ts", skip.New(m.fbHndApiPreCondErr, m.fbMidAppPreCond))

	// // page for rsa key-pair creation
	// m.fb.Get("/", statIncMetricFiberHandler(stats.IMFiberReqPathRoot),
	// 	func(c *fiber.Ctx) (e error) {
	// 		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	// 		c.Set(fiber.HeaderCacheControl, "no-cache")
	// 		return c.SendStatus(fiber.StatusOK)
	// 	})

	// // group media - middlewares
	// media.Use(m.fbMidAppFakeQuality)
	// media.Use(skip.New(m.fbMidAppBalance, m.fbMidAppBalancerLottery))

	// // group media - core cache fetcher
	// media.Use(m.fbHndApiCoreBalance)

	// // group media - sign handler
	// media.Use(m.fbHndAppRequestSign)

	// ! should be at the end of handlers list !
	// custom 404 handler for fiber.Error allocs reduce
	m.fb.Use(fiber404ErrorHandler)
}

func fiberErrorHandler(c *fiber.Ctx, err error) (_ error) {
	log := utils.ContextValueExtract[*zerolog.Logger](c.UserContext(), utils.CtxZeroLogger)

	// reject invalid requests
	if strings.TrimSpace(c.Hostname()) == "" {
		log.Warn().Msgf("invalid request from %s: %+v ; error - %+v",
			utils.IPFromFiberRequest(c), c, err)

		sts := utils.ContextValueExtract[*stats.Stats](c.UserContext(), utils.CtxStats)
		if sts != nil {
			sts.WriteIncMetric(stats.IMHTTPServerRequest)
			sts.WriteIncMetric(stats.IMHTTPServerInvalidRequest)
		}

		return c.Context().Conn().Close()
	}

	// disable caching for error responder
	c.Set(fiber.HeaderCacheControl, "no-cache")

	// JSON error content-type:
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	// * using errors.As is more readable way, yes
	// * but we building high-throughput solution, so we need allocs minimization
	// * for responding on invalid requests
	switch err := err.(type) {
	case *fiber.Error:
		writeJsonErrorFastTo(c, err.Code, err.Message)
		c.Status(err.Code)

		utils.ReleaseFiberError(err)
	default:
		writeJsonErrorFastTo(c, fiber.StatusInternalServerError, err.Error())
		c.Status(fiber.StatusInternalServerError)
	}

	// TODO : fixme; I think it can be dropped
	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		utils.Rlog(c, zerolog.DebugLevel).Msgf("%+v", err)
	}

	return
}

func (m *Service) fhttpListenerInitialization() func() error {
	reuseport, deferaccept, fastopen, backlog :=
		m.cli.Bool("http-adv-reuseport"),
		m.cli.Bool("http-adv-deferaccept"),
		m.cli.Bool("http-adv-tcpfastopen"),
		m.cli.Int("http-adv-backlog")

	if !reuseport && !deferaccept && !fastopen && backlog == 0 {
		return nil
	}

	tcpopts := &tcplisten.Config{
		ReusePort:   reuseport,
		DeferAccept: deferaccept,
		FastOpen:    fastopen,
		Backlog:     backlog,
	}

	ln, e := tcpopts.NewListener("tcp4", m.cli.String("http-listen-addr"))
	if e != nil {
		m.log.Error().Msg("could not initialize custom net.Listener, due to - " + e.Error())
		return nil
	}

	// TODO ??
	return func() error {
		return m.fb.Listener(ln)
	}
}
