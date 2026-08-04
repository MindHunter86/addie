package config

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
)

var ErrInvalidURL = errors.New("given URL is invalid")

type HttpClient struct {
	*fasthttp.HostClient
	uri *fasthttp.URI

	fdead time.Duration
	log   *zerolog.Logger
}

func NewHttpClient(c context.Context, url string) (_ *HttpClient, e error) {
	rri := fasthttp.AcquireURI()
	if e = rri.Parse(nil, utils.UnsafeBytes(url)); e != nil {
		fasthttp.ReleaseURI(rri)
		return
	}

	cc := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	return &HttpClient{
		HostClient: &fasthttp.HostClient{
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent#crawler_and_bot_ua_strings
			Name: fmt.Sprintf("Mozilla/5.0 (compatible; %s/%s; +mailto:%s)",
				cc.App.Name, cc.App.Version, cc.App.Authors[0].Email),

			// Addr:  cc.String("asmas-server-addr"),
			IsTLS: bytes.Equal(rri.Scheme(), []byte("https")),

			TLSConfig: &tls.Config{
				InsecureSkipVerify: cc.Bool("http-client-ssl-insecure"), // skipcq: GSC-G402 false-positive
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS13,
			},

			MaxConns: cc.Int("http-client-max-conns"),

			ReadTimeout:         cc.Duration("http-client-timeout-read"),
			WriteTimeout:        cc.Duration("http-client-timeout-write"),
			MaxIdleConnDuration: cc.Duration("http-client-timeout-idle"),
			MaxConnDuration:     cc.Duration("http-client-timeout-conn"),
			MaxConnWaitTimeout:  cc.Duration("http-client-timeout-conn-wait"),

			DisableHeaderNamesNormalizing: false,
			DisablePathNormalizing:        false,
			NoDefaultUserAgentHeader:      false,

			Dial: (&fasthttp.TCPDialer{
				Concurrency:      cc.Int("http-client-tcpdial-concurr"),
				DNSCacheDuration: cc.Duration("http-client-dnscache-dur"),
			}).Dial,

			// !
			// ? DialTimeout
		},

		uri: rri,

		fdead: cc.Duration("http-client-timeout-filewrite"),
		log:   utils.ContextValueExtract[*zerolog.Logger](c, utils.CtxZeroLogger),
	}, nil
}

func (m *HttpClient) downloadSourceFromURL(url, temp string) (e error) {
	if url == "" || temp == "" {
		return os.ErrNotExist
	}

	var fd *os.File
	if fd, e = os.OpenFile(temp, os.O_RDWR, 0600); e != nil {
		return utils.ExtraErrorWrapper(e, "could not prepare tmp file for source download")
	}
	defer fd.Close()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)

	req.SetURI(m.uri)

	req.Header.Set(fasthttp.HeaderAccept, fiber.MIMEOctetStream)
	req.Header.Set(fasthttp.HeaderUserAgent, m.Name)
	req.Header.Set(fasthttp.HeaderKeepAlive, "timeout=5, max=1000")
	req.Header.Set(fasthttp.HeaderConnection, "keep-alive")
	req.Header.Set(fasthttp.HeaderCacheControl, "no-cache")
	req.Header.Set(fasthttp.HeaderPragma, "no-cache")

	if e = m.Do(req, rsp); e != nil {
		return
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		m.log.Trace().Msg(req.String())
		m.log.Trace().Msg(rsp.String())
	}

	status := rsp.StatusCode()
	if status >= fasthttp.StatusMultipleChoices && status < fasthttp.StatusBadRequest {
		return errors.New("remote responded with 3XX status, canceling downloading")
	} else if status >= fasthttp.StatusBadRequest && status < fasthttp.StatusInternalServerError {
		return errors.New("remote responded with 4XX status, is URL is correct?")
	} else if status != fasthttp.StatusOK {
		return fmt.Errorf("could not perform downloading due to unexpected %d status from server", rsp.StatusCode())
	}

	if len(rsp.Body()) == 0 {
		return fmt.Errorf("remote responded with an empty body on status code 200")
	}

	fd.SetWriteDeadline(time.Now().Add(m.fdead))

	var n int
	if n, e = fd.Write(rsp.Body()); e != nil && errors.Is(e, os.ErrDeadlineExceeded) {
		return fmt.Errorf("could not perform response saving within %s, check http-client-timeout-filewrite arg", m.fdead.String())
	} else if e != nil {
		return utils.ExtraErrorWrapper(e, "response saving was failed;")
	}

	if n != rsp.Header.ContentLength() {
		m.log.Warn().Msgf("seems remote reponse saving is corrupted, file %s, remote %s", fd.Name(), m.uri.String())
	}

	return
}
