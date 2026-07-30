package service

import (
	"strings"

	"github.com/MindHunter86/addie/internal/stats"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/valyala/tcplisten"
)

func fiberErrorHandler(c *fiber.Ctx, err error) (_ error) {
	// reject invalid requests
	if strings.TrimSpace(c.Hostname()) == "" {
		gLog.Warn().Msgf("invalid request from %s: %+v ; error - %+v",
			utils.IPFromFiberRequest(c), c, err)

		sts := utils.ContextValueExtract[*stats.Stats](gCtx, utils.CtxStats)
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
	// * but we building anti-ddos solution, so we need allocs minimization
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
		gCli.Bool("http-adv-reuseport"),
		gCli.Bool("http-adv-deferaccept"),
		gCli.Bool("http-adv-tcpfastopen"),
		gCli.Int("http-adv-backlog")

	if !reuseport && !deferaccept && !fastopen && backlog == 0 {
		return nil
	}

	tcpopts := &tcplisten.Config{
		ReusePort:   reuseport,
		DeferAccept: deferaccept,
		FastOpen:    fastopen,
		Backlog:     backlog,
	}

	ln, e := tcpopts.NewListener("tcp4", gCli.String("http-listen-addr"))
	if e != nil {
		gLog.Error().Msg("could not initialize custom net.Listener, due to - " + e.Error())
		return nil
	}

	return func() error {
		return m.fb.Listener(ln)
	}
}
