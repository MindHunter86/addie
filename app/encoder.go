package app

import (
	"errors"
	"fmt"

	"github.com/k0kubun/pp"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
)

type EncoderClient struct {
	*fasthttp.HostClient

	originBase string
}

func NewEncoderClient(c *cli.Context) *EncoderClient {
	return &EncoderClient{
		originBase: c.String("encoder-origin-url"),

		HostClient: &fasthttp.HostClient{
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent#crawler_and_bot_ua_strings
			Name: fmt.Sprintf("Mozilla/5.0 (compatible; %s/%s; +https://anilibria.top/support)",
				c.App.Name, c.App.Version),

			Addr: c.String("encoder-origin-url"),

			MaxConns: c.Int("proxy-max-conns-per-host"),

			ReadTimeout:         c.Duration("proxy-read-timeout"),
			WriteTimeout:        c.Duration("proxy-write-timeout"),
			MaxIdleConnDuration: c.Duration("proxy-idle-timeout"),
			MaxConnDuration:     c.Duration("proxy-conn-timeout"),

			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			NoDefaultUserAgentHeader:      true,

			Dial: (&fasthttp.TCPDialer{
				Concurrency:      c.Int("proxy-tcpdial-concurr"),
				DNSCacheDuration: c.Duration("proxy-dns-cache-dur"),
			}).Dial,

			// !!!
			// !!!
			// !!!
			// ? DialTimeout
		},
	}
}

func (m *EncoderClient) fetchM3U8Url(url string) (body []byte, e error) {
	// pp.Println(m.originBase + url)
	// var status int
	// if status, body, e = m.Get(nil, m.originBase+url); e != nil {
	// 	return
	// }

	// if status != 200 {
	// 	return nil, errors.New("unexpected response code from encoder server")
	// }

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)

	uri.SetHost("cache.libria.fun")
	uri.SetPath(url)
	req.SetURI(uri)

	req.Header.SetMethod(fasthttp.MethodGet)

	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)

	if e = m.Do(req, rsp); e != nil {
		return
	}

	if rsp.StatusCode() != fasthttp.StatusOK {
		pp.Println(rsp.StatusCode())
		return nil, errors.New("unexpected response code from encoder server")
	}

	body = make([]byte, len(rsp.Body()))
	copy(body, rsp.Body())

	return
}
