package app

import (
	"errors"
	"math/rand"
	"strings"

	"github.com/MindHunter86/addie/balancer"
	"github.com/MindHunter86/addie/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
)

var (
	// errApiPreBadHeaders = errors.New("could not parse required headers")
	errApiPreBadUri    = errors.New("invalid uri")
	errApiPreBadId     = errors.New("invalid id")
	errApiPreBadServer = errors.New("invalid server")
	errApiPreUriRegexp = errors.New("regexp matching failure")
)

const (
	apiHeaderUri      = "X-Client-Uri"
	apiHeaderId       = "X-Client-Id"
	apiHeaderServer   = "X-Cache-Server"
	apiHeaderLocation = "X-Location"
)

type appMidError uint8

const (
	errMidAppPreHeaderUri appMidError = 1 << iota
	errMidAppPreHeaderId
	errMidAppPreHeaderServer
	errMidAppPreUriRegexp
)

// func (m *App) hlpHandler(ctx *fasthttp.RequestCtx) {
// 	// client IP parsing
// 	cip := string(ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor))
// 	if cip == "" || cip == "127.0.0.1" {
// 		rlog(ctx).Debug().Str("remote_addr", ctx.RemoteIP().String()).Str("x_forwarded_for", cip).Msg("")
// 		m.hlpRespondError(&ctx.Response, errHlpBadIp)
// 		return
// 	}
// }

func (m *App) fbMidBalanceCond(ctx *fiber.Ctx) (skip bool) {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlcPreCond)
	rlog(ctx).Trace().Interface("hdrs", ctx.GetReqHeaders()).Msg("cache-XXX-internal precond balancer")

	var errs appMidError

	gLog.Trace().Interface("hdrs", ctx.GetReqHeaders()).Msg("debug")
	switch h := ctx.GetReqHeaders(); {
	case strings.TrimSpace(h[apiHeaderUri][0]) == "":
		errs = errs | errMidAppPreHeaderUri
		ctx.Locals("errors", errs)
		return
	case strings.TrimSpace(h[apiHeaderId][0]) == "":
		errs = errs | errMidAppPreHeaderId
		ctx.Locals("errors", errs)
		return
	case strings.TrimSpace(h[apiHeaderServer][0]) == "":
		errs = errs | errMidAppPreHeaderServer
		ctx.Locals("errors", errs)
		return

		// !!!
		// !!!
		// !!!
		// !!!
		// // parse and test given chunk uri
		// huri := strings.TrimSpace(ctx.Get(apiHeaderUri))
		// if huri == "" {
		// 	errs = errs | errMidAppPreHeaderUri
		// } else if !m.chunkRegexp.Match([]byte(huri)) {
		// 	errs = errs | errMidAppPreUriRegexp
		// } else {
		// 	ctx.Locals("uri", huri)
	}

	if strings.HasPrefix(ctx.Path(), "/videos/media/ts") {
		var id, server string

		if id = strings.TrimSpace(ctx.Get(apiHeaderId)); id == "" {
			errs = errs | errMidAppPreHeaderId
		} else if server = strings.TrimSpace(ctx.Get(apiHeaderServer)); server == "" {
			errs = errs | errMidAppPreHeaderServer
		}

		ctx.Locals("srv", server)
		ctx.Locals("uid", id)
	}

	ctx.Locals("errors", errs)
	return errs == 0
}

// fake quality check
func (m *App) fbMidAppFakeQuality(ctx *fiber.Ctx) error {
	m.lapRequestTimer(ctx, utils.FbReqTmrFakeQuality)
	rlog(ctx).Trace().Msg("fake quality check")

	uri := ctx.Locals("uri").(string)
	tsr := NewTitleSerieRequest(uri)

	if !tsr.isValid() {
		ctx.Locals("uri", uri)
		return ctx.Next()
	}

	buf, ok, e := m.runtime.Config.GetValue(runtime.ConfigParamQuality)
	if !ok || e != nil {
		rlog(ctx).Warn().
			Msg("could not get lock for reading quality or softer says no; skipping fake quality chain")
		return ctx.Next()
	}

	quality := buf.(utils.TitleQuality)
	rlog(ctx).Debug().Uint16("tsr", uint16(tsr.getTitleQuality())).Uint16("coded", uint16(quality)).
		Msg("quality check")
	if tsr.getTitleQuality() <= quality {
		ctx.Locals("uri", uri)
		return ctx.Next()
	}

	// precondition finished; quality cool down
	ctx.Locals("uri", m.getUriWithFakeQuality(tsr, uri, quality))
	return ctx.Next()
}

// if return value == true - Balance() will be skipped
func (m *App) fbMidAppBalancerLottery(ctx *fiber.Ctx) bool {
	lottery, ok, e := m.runtime.Config.GetValue(runtime.ConfigParamLottery)
	if !ok || e != nil {
		rlog(ctx).Warn().Msg(e.Error())
		return !ok // always true
	}

	return lottery.(int) < rand.Intn(99)+1
}

func (m *App) fbMidAppBalance(ctx *fiber.Ctx) (e error) {
	m.lapRequestTimer(ctx, utils.FbReqTmrConsulLottery)
	rlog(ctx).Trace().Msg("consul lottery winner, rewriting destination server...")

	if e = m.balanceFiberRequest(ctx, []balancer.Balancer{m.cloudBalancer, m.bareBalancer}); e != nil {
		return
	}

	return ctx.Next()
}

// blocklist
func (m *App) fbMidAppBlocklist(ctx *fiber.Ctx) error {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlocklist)

	if buf, ok, e := m.runtime.Config.GetValue(runtime.ConfigParamBlocklist); !ok || e != nil {
		gLog.Warn().
			Msg("could not get lock for reading quality or softer says no; skipping fake quality chain")
		return ctx.Next()
	} else if buf.(int) == 0 {
		return ctx.Next()
	}

	if m.blocklist.IsExists(ctx.IP()) {
		rlog(ctx).Debug().Str("cip", ctx.IP()).Msg("client has been banned, forbid request")
		return fiber.NewError(fiber.StatusForbidden)
	}

	return ctx.Next()
}
