package app

import (
	"crypto/md5" // skipcq: GSC-G501 md5 used in nginx, no choise to fix it
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// getHlpExtra() simply is a secure_link implementation

// docs:
// https://nginx.org/ru/docs/http/ngx_http_secure_link_module.html#secure_link
//
// bash example:
//
//	echo -n '2147483647/s/link127.0.0.1 secret' | \
//		openssl md5 -binary | openssl base64 | tr +/ -_ | tr -d =
func (*App) getHlpExtra(uri, sip, uid string) (expires, extra string) {

	localts := time.Now().Local().Add(gCli.Duration("link-expiration")).Unix()
	expires = strconv.Itoa(int(localts))

	// secret link skeleton:
	// expire:uri:cache_ip:uid secret
	gLog.Trace().Strs("extra_values", []string{expires, uri, sip, uid, gCli.String("link-secret")}).Msg("")

	// concat all values
	// ?? buf := expires + uri + cip + sip + uid + " " + gCli.String("link-secret")
	buf := expires + uri + sip + uid + " " + gCli.String("link-secret")

	// md5 sum `openssl md5 -binary`
	md5sum := md5.Sum([]byte(buf)) // skipcq: GSC-G401 md5 used in nginx, no choise to fix it

	// base64 encoding `openssl base64`
	b64buf := base64.StdEncoding.EncodeToString(md5sum[:])

	// replace && trim string `tr +/ -_ | tr -d =`
	extra = strings.Trim(
		strings.ReplaceAll(
			strings.ReplaceAll(
				b64buf, "+", "-",
			),
			"/", "_",
		), "=")

	return
}
