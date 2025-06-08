package app

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
)

var (
	errApiAbnormalResponse = errors.New("there is some problems with anilibria servers communication")
)

type ApiClient struct {
	http *http.Client

	apiBaseUrl *url.URL
}

const defaultApiMethodFilter = "id,code,names,updated,last_change,player"

type ApiRequestMethod string

const (
	apiMethodGetTitle ApiRequestMethod = "/getTitle"
)

type (
	apiError struct {
		Error *apiErrorDetails
	}
	apiErrorDetails struct {
		Code    int
		Message string
	}
	apiResponse struct {
		payload []byte
		err     error
	}
)

func (m *apiResponse) Err() error {
	return m.err
}

func (m *apiResponse) Error() string {
	return m.err.Error()
}

func NewApiClient(c context.Context) (ac *ApiClient, e error) {
	cli, log :=
		utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext),
		utils.ContextValueExtract[*zerolog.Logger](c, utils.CtxZeroLogger)

	transportDialContext := func(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
		return dialer.DialContext
	}

	httpt := &http.Transport{
		DialContext: transportDialContext(&net.Dialer{
			Timeout:   cli.Duration("http-client-conn-timeout"),
			KeepAlive: cli.Duration("http-client-idle-timeout"),
		}),

		TLSHandshakeTimeout: cli.Duration("http-client-ssl-timeout"),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cli.Bool("http-client-insecure"), // skipcq: GSC-G402 false-positive
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
		},

		ResponseHeaderTimeout: cli.Duration("http-client-write-timeout"),

		MaxIdleConnsPerHost: cli.Int("http-client-max-conns-per-host"),
		IdleConnTimeout:     cli.Duration("http-client-idle-timeout"),

		DisableCompression: false,
		DisableKeepAlives:  false,
		ForceAttemptHTTP2:  true,
	}

	var http2t *http2.Transport
	if http2t, e = http2.ConfigureTransports(httpt); e != nil {
		log.Warn().Msgf("could not upgrade http transport to v2 - %s", e.Error())
	} else {
		http2t.ReadIdleTimeout = time.Second // ping is performed at N whenever no frame was received in the meantime
		http2t.PingTimeout = 3 * time.Second
	}

	ac = &ApiClient{
		http: &http.Client{
			Timeout:   cli.Duration("http-client-conn-timeout") + cli.Duration("http-client-read-timeout"),
			Transport: httpt,
		},
	}

	return ac, ac.getApiBaseUrl()
}

func (m *ApiClient) getApiBaseUrl() (e error) {
	m.apiBaseUrl, e = url.Parse(gCli.String("anilibria-api-baseurl"))
	return e
}
