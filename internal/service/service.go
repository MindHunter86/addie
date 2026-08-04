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

	"github.com/MindHunter86/addie/internal/app"
	"github.com/MindHunter86/addie/internal/stats"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type Service struct {
	fb *fiber.App
	wg sync.WaitGroup

	abort context.CancelFunc

	log *zerolog.Logger
	cli *cli.Context
	ctx context.Context
}

func NewService(c *cli.Context, l, al *zerolog.Logger) *Service {
	service := &Service{
		fb: fiber.New(fiber.Config{
			EnableTrustedProxyCheck: c.String("http-trusted-proxies") != "",
			TrustedProxies:          strings.Split(c.String("http-trusted-proxies"), ","),
			ProxyHeader:             c.String("http-realip-header"),

			DisableStartupMessage: true,

			AppName:      c.App.Name,
			ServerHeader: fmt.Sprintf("%s/%s", c.App.Name, c.App.Version),

			StrictRouting:      true,
			DisableDefaultDate: false,
			DisableKeepalive:   false,

			DisableHeaderNormalizing:     false,
			DisableDefaultContentType:    true,
			DisablePreParseMultipartForm: true,

			Prefork:      c.Bool("http-prefork"),
			IdleTimeout:  c.Duration("http-timeout-idle"),
			ReadTimeout:  c.Duration("http-timeout-read"),
			WriteTimeout: c.Duration("http-timeout-write"),

			Concurrency: c.Int("http-concurrency-conns"),

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

		log: l,
		cli: c,
	}

	service.ctx, service.abort = context.WithCancel(context.Background())
	service.ctx = context.WithValue(service.ctx, utils.CtxZeroLogger, l)
	service.ctx = context.WithValue(service.ctx, utils.CtxCliContext, c)
	service.ctx = context.WithValue(service.ctx, utils.CtxAccsLogger, al)

	return service
}

func (m *Service) Bootstrap() (e error) {
	// PREBOOTSTRAP SECTION:
	//
	// GC tunning
	gogc := m.cli.Int("runtime-gogc")
	oldgc := debug.SetGCPercent(gogc)
	m.log.Info().Msgf("setting GOGC from %d to %d", oldgc, gogc)

	//
	// prepare all subservices
	if e = utils.ForEachSubservice(func(ck utils.ContextKey, sh utils.SubserviceHandler) error {
		m.log.Trace().Msgf("prepare %s subservice...", utils.CKtoa[ck])
		defer m.log.Trace().Msgf("%s subservice has been prepared", utils.CKtoa[ck])

		if val, err := sh(m.ctx); err == nil {
			m.ctx = context.WithValue(m.ctx, ck, val)
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
		return utils.ExtraErrorWrapper(cb(m.ctx), "on-service-bootstrap callback run")
	}); e != nil {
		return
	}

	//
	// LEGACY section (will be refactored)

	// fiber configuration
	// TODO : temporary disable svc router
	// m.fiberMiddlewareInitialization()
	// m.fiberRouterInitialization()
	app := utils.ContextValueExtract[*app.App](m.ctx, utils.CtxApp)
	app.HandleRoutes(m.fb)

	// custom listener configuration
	flisten := func() error {
		return m.fb.Listen(m.cli.String("http-listen-addr"))
	}
	if fn := m.fhttpListenerInitialization(); fn != nil {
		m.log.Info().Msg("configuring custom fasthttp net.listener...")
		flisten = fn
	}

	// http server bootstrap (should be at the end of bootstrap)
	utils.Go(&m.wg, m.log, func() {
		m.log.Debug().Msg("starting fiber http server...")
		defer m.log.Debug().Msg("fiber http server has been stopped")

		if err := flisten(); errors.Is(err, context.Canceled) {
			return
		} else if err != nil {
			m.log.Error().Err(err).Msg("fiber internal error")
			m.abort()
		}
	})

	// main event loop
	utils.Go(&m.wg, m.log, m.loop)
	m.log.Info().Msg("all subservices were started, waiting for waitgroup...")

	// destructor
	m.wg.Wait()
	return m.destruct(e)
}

func (m *Service) destruct(e error) error {
	_ = utils.CallCallbacks(utils.OnServiceDestruct, func(cb utils.ServiceCallback) error {
		if err := cb(m.ctx); err != nil {
			m.log.Warn().Msg(utils.ExtraErrorWrapper(err, "on-service-destruct callback call").Error())
		}
		return nil
	})

	if m.log.GetLevel() <= zerolog.DebugLevel {
		// brief delay to ensure all subservices complete
		// and print correct information about goroutines
		time.Sleep(250 * time.Millisecond)
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	}

	return e
}

func (m *Service) loop() {
	m.log.Debug().Msg("starting main event loop...")
	defer m.log.Debug().Msg("main event loop has been stopped")

	sts := utils.ContextValueExtract[*stats.Stats](m.ctx, utils.CtxStats)

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
				m.ctx, utils.CtxTickerTick, tick), utils.CtxTickerLap, lap))
	}
	logIfError := func(e error) {
		if e != nil {
			m.log.Warn().Msg(e.Error())
		}
	}

	m.log.Info().Msg("application ready for serving requests")

LOOP:
	for {
		select {
		// TERM signals
		case <-kernQuitSignal:
			m.abort()
		case <-m.ctx.Done():
			m.log.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP

		// DEBUG signals
		case <-kernDumpSignal:
			m.log.Debug().Msg("kernel signal has been caught; dumping goroutines into stdout...")
			debug.PrintStack()
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

			sts.WriteIncMetric(stats.IMSvcKernSignDebug)

		// tickers
		case lap := <-loopticker.C:
			tick++
			handleTickerTick(lap, func(ctx context.Context) {
				utils.GoCallCallbacks(utils.OnServiceTicker1sec, &m.wg, m.log, func(cb utils.ServiceCallback) {
					logIfError(utils.ExtraErrorWrapper(cb(ctx), "on-service-ticker-1sec callback run"))
				})
			})
		}
	}

	// http destruct (wtf fiber?)
	// ShutdownWithContext() may be called only after fiber.Listen is running (O_o)
	if e := m.fb.ShutdownWithContext(m.ctx); e != nil {
		m.log.Error().Err(e).Msg("fiber Shutdown() error")
	}
}
