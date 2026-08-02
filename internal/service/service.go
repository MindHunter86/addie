package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/MindHunter86/addie/internal/runtime"
	"github.com/MindHunter86/addie/internal/stats"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// todo : delete global variables, move to Context
var (
	gCli *cli.Context
	gLog *zerolog.Logger

	gCtx context.Context
)

type Service struct {
	fb *fiber.App
	wg sync.WaitGroup

	abort context.CancelFunc

	runtime *runtime.Runtime
}

func NewService(c *cli.Context, l, al *zerolog.Logger) *Service {
	gCli, gLog = c, l

	service := &Service{
		fb: fiber.New(fiber.Config{
			EnableTrustedProxyCheck: gCli.String("http-trusted-proxies") != "",
			TrustedProxies:          strings.Split(gCli.String("http-trusted-proxies"), ","),
			ProxyHeader:             gCli.String("http-realip-header"),

			DisableStartupMessage: true,

			AppName:      c.App.Name,
			ServerHeader: fmt.Sprintf("%s/%s", c.App.Name, c.App.Version),

			StrictRouting:      true,
			DisableDefaultDate: false,
			DisableKeepalive:   false,

			DisableHeaderNormalizing:     true,
			DisableDefaultContentType:    true,
			DisablePreParseMultipartForm: true,

			Prefork:      gCli.Bool("http-prefork"),
			IdleTimeout:  gCli.Duration("http-idle-timeout"),
			ReadTimeout:  gCli.Duration("http-read-timeout"),
			WriteTimeout: gCli.Duration("http-write-timeout"),

			Concurrency: gCli.Int("http-concurrency-conns"),

			BodyLimit:      1 << 16, // 64KiB
			ReadBufferSize: 1 << 13, // 8KiB

			// GET + HEAD
			GETOnly: false,
			RequestMethods: []string{
				fiber.MethodHead,
				fiber.MethodGet,
				fiber.MethodOptions,
				fiber.MethodPost,
			},

			ErrorHandler: fiberErrorHandler,

			// JSONEncoder: easyjson.Marshal,
			// JSONDecoder: easyjson.Unmarshal,

			// todo : we need fasthttp.MaxConnsPerIP
		}),
	}

	gCtx, service.abort = context.WithCancel(context.Background())
	gCtx = context.WithValue(gCtx, utils.CtxZeroLogger, gLog)
	gCtx = context.WithValue(gCtx, utils.CtxCliContext, gCli)
	gCtx = context.WithValue(gCtx, utils.CtxAccsLogger, al)

	return service
}

func (m *Service) Bootstrap() (e error) {
	// PREBOOTSTRAP SECTION:
	//
	// GC tunning
	gogc := gCli.Int("runtime-gogc")
	oldgc := debug.SetGCPercent(gogc)
	gLog.Info().Msgf("setting GOGC from %d to %d", oldgc, gogc)

	//
	// prepare all subservices
	if e = utils.ForEachSubservice(func(ck utils.ContextKey, sh utils.SubserviceHandler) error {
		gLog.Trace().Msgf("prepare %s subservice...", utils.CKtoa[ck])
		defer gLog.Trace().Msgf("%s subservice has been prepared", utils.CKtoa[ck])

		if val, err := sh(gCtx); err == nil {
			gCtx = context.WithValue(gCtx, ck, val)
			return nil
		} else {
			return err
		}
	}); e != nil {
		return
	}

	//
	// BOOTSTRAP SECTION:
	if e = utils.CallCallbacks(utils.OnServiceBootstrap, func(cb utils.ServiceCallback) error {
		return utils.ExtraErrorWrapper(cb(gCtx), "on-service-bootstrap callback run")
	}); e != nil {
		return
	}

	// fiber configuration
	m.fiberMiddlewareInitialization()
	m.fiberRouterInitialization()

	// custom listener configuration
	flisten := func() error {
		return m.fb.Listen(gCli.String("http-listen-addr"))
	}
	if fn := m.fhttpListenerInitialization(); fn != nil {
		gLog.Info().Msg("configuring custom fasthttp net.listener...")
		flisten = fn
	}

	// http server bootstrap (should be at the end of bootstrap)
	utils.Go(&m.wg, gLog, func() {
		gLog.Debug().Msg("starting fiber http server...")
		defer gLog.Debug().Msg("fiber http server has been stopped")

		if err := flisten(); errors.Is(err, context.Canceled) {
			return
		} else if err != nil {
			gLog.Error().Err(err).Msg("fiber internal error")
			m.abort()
		}
	})

	// main event loop
	utils.Go(&m.wg, gLog, m.loop)
	gLog.Info().Msg("all subservices were started, waiting for waitgroup...")

	// destructor
	m.wg.Wait()
	return m.destruct(e)
}

func (*Service) destruct(e error) error {
	_ = utils.CallCallbacks(utils.OnServiceDestruct, func(cb utils.ServiceCallback) error {
		if err := cb(gCtx); err != nil {
			gLog.Warn().Msg(utils.ExtraErrorWrapper(err, "on-service-destruct callback call").Error())
		}
		return nil
	})

	if gLog.GetLevel() <= zerolog.DebugLevel {
		// brief delay to ensure all subservices complete
		// and print correct information about goroutines
		time.Sleep(250 * time.Millisecond)
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	}

	return e
}

func (m *Service) loop() {
	gLog.Debug().Msg("starting main event loop...")
	defer gLog.Debug().Msg("main event loop has been stopped")

	sts := utils.ContextValueExtract[*stats.Stats](gCtx, utils.CtxStats)

	// debug does not work on windows systems
	kernDumpSignal := m.listenForDebugSignal()
	kernQuitSignal := make(chan os.Signal, 1)
	signal.Notify(kernQuitSignal, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGABRT)

	m.wg.Add(1)
	defer m.wg.Done()

	loopticker, tick := time.NewTicker(time.Second), uint64(1)
	defer loopticker.Stop()

	// loop helpers
	handleTickerTick := func(lap time.Time, fn func(context.Context)) {
		fn(context.WithValue(
			context.WithValue(
				gCtx, utils.CtxTickerTick, tick), utils.CtxTickerLap, lap))
	}
	logIfError := func(e error) {
		if e != nil {
			gLog.Warn().Msg(e.Error())
		}
	}

	gLog.Info().Msg("application ready for serving requests")

LOOP:
	for {
		select {
		// TERM signals
		case <-kernQuitSignal:
			m.abort()
		case <-gCtx.Done():
			gLog.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP

		// DEBUG signals
		case <-kernDumpSignal:
			gLog.Debug().Msg("kernel signal has been caught; dumping goroutines into stdout...")
			debug.PrintStack()
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

			sts.WriteIncMetric(stats.IMSvcKernSignDebug)

		// tickers
		case lap := <-loopticker.C:
			tick++
			handleTickerTick(lap, func(ctx context.Context) {
				utils.GoCallCallbacks(utils.OnServiceTicker1sec, &m.wg, gLog, func(cb utils.ServiceCallback) {
					logIfError(utils.ExtraErrorWrapper(cb(ctx), "on-service-ticker-1sec callback run"))
				})
			})
		}
	}

	// http destruct (wtf fiber?)
	// ShutdownWithContext() may be called only after fiber.Listen is running (O_o)
	if e := m.fb.ShutdownWithContext(gCtx); e != nil {
		gLog.Error().Err(e).Msg("fiber Shutdown() error")
	}
}
