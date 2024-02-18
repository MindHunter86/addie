package app

import (
	"errors"
	"net/url"

	"github.com/MindHunter86/addie/balancer"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
)

var (
	errFbApiInvalidMode    = errors.New("mode argument is invalid; values soft, hard are permited only")
	errFbApiInvalidQuality = errors.New("quality argument is invalid; 480, 720, 1080 values are permited only")
)

func (*App) fbHndApiPreCondErr(ctx *fiber.Ctx) error {
	switch ctx.Locals("errors").(appMidError) {
	case errMidAppPreHeaderUri:
		rlog(ctx).Warn().Msg(errApiPreBadUri.Error())
		ctx.Set("X-Error", errApiPreBadUri.Error())
		ctx.SendString(errApiPreBadUri.Error())
	case errMidAppPreHeaderId:
		rlog(ctx).Warn().Msg(errApiPreBadId.Error())
		ctx.Set("X-Error", errApiPreBadId.Error())
		ctx.SendString(errApiPreBadId.Error())
	case errMidAppPreHeaderServer:
		rlog(ctx).Warn().Msg(errApiPreBadServer.Error())
		ctx.Set("X-Error", errApiPreBadServer.Error())
		ctx.SendString(errApiPreBadServer.Error())
	case errMidAppPreUriRegexp:
		rlog(ctx).Warn().Msg(errApiPreUriRegexp.Error())
		ctx.Set("X-Error", errApiPreUriRegexp.Error())
		ctx.SendString(errApiPreUriRegexp.Error())
	default:
		rlog(ctx).Warn().Msg("unknown error")
	}

	return ctx.SendStatus(fiber.StatusPreconditionFailed)
}

func (m *App) fbHndAppRequestSign(ctx *fiber.Ctx) (e error) {
	m.lapRequestTimer(ctx, utils.FbReqTmrReqSign)
	rlog(ctx).Trace().Msg("new 'sign request' request")

	srv, uri := ctx.Locals("srv").(string), ctx.Locals("uri").(string)
	expires, extra := m.getHlpExtra(
		uri,
		srv,
		ctx.Locals("uid").(string),
	)

	var rrl *url.URL
	if rrl, e = url.Parse(srv + uri); e != nil {
		rlog(ctx).Debug().Str("url_parse", srv+uri).Str("remote_addr", ctx.IP()).
			Msg("could not sign request; url.Parse error")
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	var rgs = &url.Values{}
	rgs.Add("expires", expires)
	rgs.Add("extra", extra)
	rrl.RawQuery, rrl.Scheme = rgs.Encode(), "https"

	rlog(ctx).Debug().Str("computed_request", rrl.String()).Str("remote_addr", ctx.IP()).
		Msg("request signing completed")
	ctx.Set(apiHeaderLocation, rrl.String())
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndBlcNodesBalance(ctx *fiber.Ctx) error {
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

	rlog(ctx).Trace().Msg("im here!")

	err := m.balanceFiberRequest(ctx, []balancer.Balancer{m.bareBalancer})

	var ferr *fiber.Error
	if errors.As(err, &ferr) {
		rlog(ctx).Trace().Msgf("im here! %d %s", ferr.Code, ferr.Message)
		// ! error here
		return err
	} else if err != nil {
		// ! undefined error here
		panic("undefined error in BM balancer")
	}

	ctx.Set("X-Location", ctx.Locals("srv").(string))
	return ctx.SendStatus(fiber.StatusNoContent)
}
