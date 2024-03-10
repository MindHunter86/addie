package app

import (
	// skipcq: GSC-G501 md5 used in nginx, no choise to fix it

	"bytes"
	"crypto/md5"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/MindHunter86/addie/runtime"
	"github.com/gofiber/fiber/v2"
	futils "github.com/gofiber/fiber/v2/utils"
	"github.com/rs/zerolog"
)

// getHlpExtra() simply is a secure_link implementation

// docs:
// https://nginx.org/ru/docs/http/ngx_http_secure_link_module.html#secure_link
//
// bash example:
//
//	echo -n '2147483647/s/link127.0.0.1 secret' | \
//		openssl md5 -binary | openssl base64 | tr +/ -_ | tr -d =
//
// Optimization faq - habr.com/ru/companies/kaspersky/articles/591725/
func (m *App) getHlpExtra(c *fiber.Ctx, uri, sip, uid []byte) (expires, _ []byte) {

	localts := time.Now().Local().Add(gCli.Duration("link-expiration")).Unix()
	expires = futils.UnsafeBytes(strconv.FormatInt(localts, 10))

	var md5buf [md5.Size]byte
	md5sum, baselen := md5.New(), base64.StdEncoding.EncodedLen(md5.Size)

	// TODO: refactor
	if m.runtime.Config.Get(runtime.ParamAccessLevel).(zerolog.Level) == zerolog.TraceLevel {
		rlog(c).Trace().
			Strs("extra_values", []string{
				futils.UnsafeString(expires),
				futils.UnsafeString(uri),
				futils.UnsafeString(sip),
				futils.UnsafeString(uid),
				gCli.String("link-secret")}).Msg("")
	}

	md5sum.Write(expires)
	md5sum.Write(uri)
	md5sum.Write(sip)
	md5sum.Write(uid)
	md5sum.Write([]byte(" "))
	md5sum.Write(futils.UnsafeBytes(gCli.String("link-secret")))

	var basebuf bytes.Buffer
	basebuf.Grow(baselen)
	for i := 0; i < baselen; i++ {
		basebuf.WriteByte(0)
	}

	base64.StdEncoding.Encode(basebuf.Bytes(), md5sum.Sum(md5buf[:0]))

	e1 := bytes.ReplaceAll(basebuf.Bytes(), []byte("+"), []byte("-"))
	e2 := bytes.ReplaceAll(e1, []byte("/"), []byte("_"))

	return expires, bytes.Trim(e2, "=")

	// secret link skeleton:
	// expire:uri:cache_ip:uid secret

	// rlog(c).Trace().
	// 	Strs("extra_values", []string{string(expires),
	// 		string(uri), string(sip), string(uid), gCli.String("link-secret")}).Msg("")

	// concat all values
	// ?? buf := expires + uri + cip + sip + uid + " " + gCli.String("link-secret")

	// bufs := expires + uri + sip + uid + " " + gCli.String("link-secret")

	// md5 sum `openssl md5 -binary`

	// md5sum := md5.Sum([]byte(bufs)) // skipcq: GSC-G401 md5 used in nginx, no choise to fix it

	// base64 encoding `openssl base64`

	// b64buf := base64.StdEncoding.EncodeToString(md5sum[:])

	// replace && trim string `tr +/ -_ | tr -d =`

	// extra = strings.Trim(
	// 	strings.ReplaceAll(
	// 		strings.ReplaceAll(
	// 			b64buf, "+", "-",
	// 		),
	// 		"/", "_",
	// 	), "=")

	// return
}
