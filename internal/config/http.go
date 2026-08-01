package config

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
)

var ErrInvalidURL = errors.New("given URL is invalid")

type HttpClient struct {
	*fasthttp.HostClient
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
	}, nil
}
