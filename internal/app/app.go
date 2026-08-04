package app

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/MindHunter86/addie/internal/balancer"
	"github.com/MindHunter86/addie/internal/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

var (
	gCli *cli.Context
	gLog *zerolog.Logger

	gCtx context.Context

	// gConsul *consulClient

	gAniApi *ApiClient

	gController *Controller
)

type App struct {
	fb    *fiber.App
	abort context.CancelFunc

	cache   *CachedTitlesBucket
	runtime *runtime.Runtime

	cloudBalancer balancer.Balancer
	bareBalancer  balancer.Balancer

	chunkRegexp *regexp.Regexp

	syslogWriter io.Writer
}

func NewApp(c *cli.Context, l *zerolog.Logger, s io.Writer) (app *App) {
	gCli, gLog = c, l

	app = &App{}
	app.syslogWriter = s

	app.fb = fiber.New(fiber.Config{
		EnableTrustedProxyCheck: gCli.String("http-trusted-proxies") == "",
		TrustedProxies:          strings.Split(gCli.String("http-trusted-proxies"), ","),
		ProxyHeader:             fiber.HeaderXForwardedFor,

		AppName:               gCli.App.Name,
		ServerHeader:          gCli.App.Name,
		DisableStartupMessage: true,

		StrictRouting:      true,
		DisableDefaultDate: true,
		DisableKeepalive:   false,

		Prefork:      gCli.Bool("http-prefork"),
		IdleTimeout:  300 * time.Second,
		ReadTimeout:  1000 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,

		DisableDefaultContentType: true,

		RequestMethods: []string{
			fiber.MethodHead,
			fiber.MethodGet,
			fiber.MethodOptions,
			fiber.MethodPost,
		},
	})

	// api controller init
	gController = NewController()

	// router configuration
	app.fiberConfigure()
	return app
}

func (m *App) Bootstrap() (e error) {
	var wg sync.WaitGroup
	var echan = make(chan error, 32)

	// goroutine helper
	gofunc := func(w *sync.WaitGroup, p func()) {
		w.Add(1)

		go func(done, payload func()) {
			payload()
			done()
		}(w.Done, p)
	}

	gCtx, m.abort = context.WithCancel(context.Background())
	gCtx = context.WithValue(gCtx, utils.ContextKeyLogger, gLog)
	gCtx = context.WithValue(gCtx, utils.ContextKeyCliContext, gCli)

	// defer m.checkErrorsBeforeClosing(echan)
	// defer wg.Wait() // !!
	defer gLog.Debug().Msg("waiting for opened goroutines")
	defer m.abort()

	// BOOTSTRAP SECTION:
	// common
	const chunksplit = `^(\/[^\/]+\/[^\/]+\/[^\/]+\/)([^\/]+)\/([^\/]+)\/([^\/]+)\/([^.\/]+)\.ts$`
	m.chunkRegexp = regexp.MustCompile(chunksplit)

	// anilibria API
	gLog.Info().Msg("starting anilibria api client...")
	if gAniApi, e = NewApiClient(); e != nil {
		return
	}

	// fake quality cooler cache
	gLog.Info().Msg("starting fake quality cache buckets...")
	m.cache = NewCachedTitlesBucket()

	// runtime
	if m.runtime, e = runtime.NewRuntime(gCtx); e != nil {
		return
	}

	// balancer V2
	gLog.Info().Msg("bootstrap balancer_v2 subsystems...")

	m.bareBalancer = balancer.NewClusterBalancer(gCtx, balancer.BalancerClusterNodes)
	m.cloudBalancer = balancer.NewClusterBalancer(gCtx, balancer.BalancerClusterCloud)

	var bservers, cservers []string
	if bservers, e = m.bareBalancer.GetFQDNsByBrace(gCli.String("balancer-node-servers")); e != nil {
		return
	}
	if cservers, e = m.cloudBalancer.GetFQDNsByBrace(gCli.String("balancer-cloud-servers")); e != nil {
		return
	}

	m.bareBalancer.UpdateServersByFQDN(bservers)
	m.cloudBalancer.UpdateServersByFQDN(cservers)

	// update API controller after balancers initialization
	gController.WithContext(gCtx, m.bareBalancer, m.cloudBalancer)
	gController.SetReady()

	// http
	gofunc(&wg, func() {
		gLog.Debug().Msg("starting fiber http server...")
		defer gLog.Debug().Msg("fiber http server has been stopped")

		if e = m.fb.Listen(gCli.String("http-listen-addr")); errors.Is(e, context.Canceled) {
			return
		} else if e != nil {
			gLog.Error().Err(e).Msg("fiber internal error")
		}
	})

	// another subsystems
	// ...

	// main event loop
	wg.Add(1)
	go m.loop(echan, wg.Done)

	gLog.Info().Msg("ready...")
	wg.Wait()
	return
}

func (m *App) loop(_ chan error, done func()) {
	defer done()

	kernSignal := make(chan os.Signal, 1)
	signal.Notify(kernSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTERM, syscall.SIGQUIT)

	gLog.Debug().Msg("initiate main event loop...")
	defer gLog.Debug().Msg("main event loop has been closed")

LOOP:
	for {
		select {
		case <-kernSignal:
			gLog.Info().Msg("kernel signal has been caught; initiate application closing...")
			m.abort()
			break LOOP
		// case err := <-errs:
		// 	gLog.Error().Err(err).Msg("there are internal errors from one of application submodule")
		// 	gLog.Info().Msg("calling abort()...")
		// 	gAbort()
		case <-gCtx.Done():
			gLog.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP
		}
	}

	// http destruct (wtf fiber?)
	// ShutdownWithContext() may be called only after fiber.Listen is running (O_o)
	if e := m.fb.ShutdownWithContext(gCtx); e != nil {
		gLog.Error().Err(e).Msg("fiber Shutdown() error")
	}
}

func (m *App) rsyslog(c *fiber.Ctx) (l *zerolog.Logger) {
	return c.Locals("syslogger").(*zerolog.Logger)
}

func rlog(c *fiber.Ctx) *zerolog.Logger {
	return c.Locals("logger").(*zerolog.Logger)
}
