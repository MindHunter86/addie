package app

import (
	"bytes"
	"errors"
	"math/rand"

	"github.com/MindHunter86/addie/balancer"
	"github.com/MindHunter86/addie/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
	futils "github.com/gofiber/fiber/v2/utils"
	"github.com/rs/zerolog"
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

func (m *App) fbMidAppPreCond(ctx *fiber.Ctx) (_ bool) {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlcPreCond)

	// TODO: refactor
	if m.runtime.Config.Get(runtime.ParamAccessLevel).(zerolog.Level) == zerolog.TraceLevel {
		rlog(ctx).Trace().Interface("hdrs", ctx.GetReqHeaders()).Msg("cache-XXX-internal precond balancer")
	}

	var u, i, d []byte
	var peekCtxHeader = ctx.Context().Request.Header.Peek

	if u = bytes.TrimSpace(peekCtxHeader(apiHeaderUri)); len(u) == 0 {
		ctx.Locals("errors", errMidAppPreHeaderUri)
		return
	} else if !m.chunkRegexp.Match(u) {
		ctx.Locals("errors", errMidAppPreUriRegexp)
		return
	}
	if i = bytes.TrimSpace(peekCtxHeader(apiHeaderId)); len(i) == 0 {
		ctx.Locals("errors", errMidAppPreHeaderId)
		return
	}

	if d = bytes.TrimSpace(peekCtxHeader(apiHeaderServer)); len(d) == 0 {
		ctx.Locals("errors", errMidAppPreHeaderServer)
		return
	}

	ctx.Locals("uri", u)
	ctx.Locals("uid", i)
	ctx.Locals("srv", d)
	return true
}

// fake quality check
func (m *App) fbMidAppFakeQuality(ctx *fiber.Ctx) error {
	m.lapRequestTimer(ctx, utils.FbReqTmrFakeQuality)
	rlog(ctx).Trace().Msg("fake quality check")

	uri := ctx.Locals("uri").([]byte)
	tsr := NewTitleSerieRequest(string(uri))
	// !!!
	// !!!
	// !!!

	if !tsr.isValid() {
		return ctx.Next()
	}

	quality := m.runtime.Config.Get(runtime.ParamQuality).(utils.TitleQuality)
	rlog(ctx).Debug().Uint16("tsr", uint16(tsr.getTitleQuality())).Uint16("coded", uint16(quality)).
		Msg("quality check")

	if tsr.getTitleQuality() <= quality {
		return ctx.Next()
	}

	// precondition finished; quality cool down
	// !!!
	// !!!
	// !!!
	ctx.Locals("uri", []byte(m.getUriWithFakeQuality(ctx, tsr, string(uri), quality)))
	return ctx.Next()
}

// if return value == true - Balance() will be skipped
func (m *App) fbMidAppBalancerLottery(_ *fiber.Ctx) bool {
	return m.runtime.Config.Get(runtime.ParamLottery).(int) < rand.Intn(99)+1 // skipcq: GSC-G404 math/rand is enough
}

func (m *App) fbMidAppBalance(ctx *fiber.Ctx) (e error) {
	m.lapRequestTimer(ctx, utils.FbReqTmrConsulLottery)
	rlog(ctx).Trace().Msg("consul lottery winner, rewriting destination server...")

	var server *balancer.BalancerServer
	uri := ctx.Locals("uri").([]byte)

	sub := m.chunkRegexp.FindSubmatch(uri)
	prefixbuf := bytes.NewBuffer(sub[utils.ChunkTitleId])
	prefixbuf.Write(sub[utils.ChunkEpisodeId])
	prefixbuf.Write(sub[utils.ChunkQualityLevel])

	// chunkname, prefix := string(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkName]), prefixbuf.String()

	// for _, cluster := range []balancer.Balancer{m.cloudBalancer, m.bareBalancer} {
	// 	// TODO
	// 	// ? do we need the failover with RandomBalancing ???
	// 	// var fallback bool

	// 	// get all servers for balancing
	// 	var status *balancer.Status
	// 	if e = cluster.Balance(chunkname, prefix); e == nil {
	// 		rlog(ctx).Error().Msg("there is no status with payload and error from balancer")
	// 	}

	// 	if errors.As(e, &status) {
	// 		if e = status.Err(); e != nil {
	// 			rlog(ctx).Error().Err(e).Interface("cluster", status.Cluster()).Msg(status.Descr())
	// 			continue
	// 		}
	// 	} else {
	// 		rlog(ctx).Error().Err(e).Msg("undefined error from balancer")
	// 	}

	// 	// parse given servers
	// 	for _, server := range status.Servers {
	// 		// if all ok (if no errors) - save destination and go to the next fiber handler:
	// 		ctx.Locals("srv",
	// 			strings.ReplaceAll(server.Name, "-node", "")+"."+gCli.String("consul-entries-domain"))

	// 		return ctx.Next()
	// 	}
	// }

	for _, cluster := range []balancer.Balancer{m.cloudBalancer, m.bareBalancer} {
		var fallback bool

		for fails := 0; fails <= gCli.Int("balancer-server-max-fails"); fails++ {

			// so if fails limit reached - use new cluster or fallback to baremetal random balancing
			if fails == gCli.Int("balancer-server-max-fails") {
				if fallback {
					rlog(ctx).Error().Str("cluster", cluster.GetClusterName()).
						Msg("internal balancer error; too many balance errors; using fallback func()...")
					return m.fbMidAppBalanceFallback(ctx)
				} else {
					fallback = true
					rlog(ctx).Error().Str("cluster", cluster.GetClusterName()).
						Msg("internal balancer error; too many balance errors; using next cluster...")
					break
				}
			}

			// trying to balance with giver cluster
			_, server, e = cluster.BalanceByChunk(prefixbuf, sub[utils.ChunkName])

			if errors.Is(e, balancer.ErrServerUnavailable) {
				rlog(ctx).Trace().Err(e).Int("fails", fails).
					Str("cluster", cluster.GetClusterName()).Msg("trying to roll new server...")
				continue
			} else if errors.Is(e, balancer.ErrUpstreamUnavailable) {
				rlog(ctx).Trace().Err(e).Int("fails", fails).Msg("temporary upstream error")
				continue
			} else if e != nil {
				rlog(ctx).Error().Err(e).
					Str("cluster", cluster.GetClusterName()).Msg("could not balance; undefined error")
				break
			}

			// if all ok (if no errors) - save destination and go to the next fiber handler:
			name := bytes.Trim(futils.UnsafeBytes(server.Name), "-node")
			domain := futils.UnsafeBytes(gCli.String("consul-entries-domain"))

			ctx.Locals("srv", utils.JoinSize(len(name)+1+len(domain), name, []byte("."), domain))
			return ctx.Next()
		}
	}

	// if we here - no alive balancers, so return error
	return fiber.NewError(fiber.StatusInternalServerError, e.Error())
}

func (m *App) fbMidAppBalanceFallback(ctx *fiber.Ctx) error {
	server, e := m.getServerFromRandomBalancer(ctx)
	if e != nil {
		return e
	}

	name := bytes.Trim(futils.UnsafeBytes(server.Name), "-node")
	domain := futils.UnsafeBytes(gCli.String("consul-entries-domain"))

	ctx.Locals("srv", utils.JoinSize(len(name)+1+len(domain), name, []byte("."), domain))
	return ctx.Next()
}

// blocklist
func (m *App) fbMidAppBlocklist(ctx *fiber.Ctx) error {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlocklist)

	if m.runtime.Config.Get(runtime.ParamBlocklist).(int) == 0 {
		return ctx.Next()
	}

	if m.blocklist.IsExists(ctx.IP()) {
		rlog(ctx).Debug().Str("cip", ctx.IP()).Msg("client has been banned, forbid request")
		return fiber.NewError(fiber.StatusForbidden)
	}

	return ctx.Next()
}

// balancer api
func (m *App) fbMidBlcPreCond(ctx *fiber.Ctx) (_ bool) {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlcPreCond)

	// TODO: refactor
	if m.runtime.Config.Get(runtime.ParamAccessLevel).(zerolog.Level) == zerolog.TraceLevel {
		rlog(ctx).Trace().Interface("hdrs", ctx.GetReqHeaders()).Msg("cache-node-internal balancer")
	}

	var u []byte
	var peekCtxHeader = ctx.Context().Request.Header.Peek

	if u = bytes.TrimSpace(peekCtxHeader(apiHeaderUri)); len(u) == 0 {
		ctx.Locals("errors", errMidAppPreHeaderUri)
		return
	} else if !m.chunkRegexp.Match(u) {
		ctx.Locals("errors", errMidAppPreUriRegexp)
		return
	}

	ctx.Locals("uri", u)
	return true
	// ctx.Locals("errors", errs)
	// return errs == 0
}
