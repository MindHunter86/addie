package utils

import (
	"bytes"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/valyala/bytebufferpool"
)

const HTTPAccessLogLevel = zerolog.InfoLevel

var nilLogger = zerolog.New(nil)

var fbErrorPool = sync.Pool{New: func() any {
	return &fiber.Error{}
}}

func AcquireFiberError(c int, m string) (fe *fiber.Error) {
	fe = fbErrorPool.Get().(*fiber.Error)
	fe.Code, fe.Message = c, m
	return
}

func ReleaseFiberError(fe *fiber.Error) {
	fbErrorPool.Put(fe)
}

func Rlog(c *fiber.Ctx, lvl zerolog.Level) *zerolog.Event {
	if !syslogOverload.TryRLock() {
		return nilLogger.Trace()
	}
	syslogOverload.RUnlock()

	ctx := c.UserContext()
	if ctx == nil {
		return nilLogger.Trace()
	}

	if l := ContextValueExtract[*zerolog.Logger](ctx, CtxAccsLogger); l != nil {
		return l.WithLevel(lvl).Uint64("id", c.Context().ID())
	}

	return nilLogger.Trace()
}

// Small dirty function for the fastest access logging
// Because we must provide logging without allocations, it's
// uses hacks for it's correct work
// Due to a huge amount of allocations in zerolog.(*Event).caller
// use it in critical methods or functions
func RlogFast(c *fiber.Ctx, lvl zerolog.Level, status int, lap time.Duration, msg string) {
	if !syslogOverload.TryRLock() {
		return
	}
	syslogOverload.RUnlock()

	ctx := c.UserContext()
	if ctx == nil {
		return
	}

	var l *zerolog.Logger
	if l = ContextValueExtract[*zerolog.Logger](ctx, CtxAccsLogger); l == nil {
		return
	}

	if l.GetLevel() > lvl || lvl == zerolog.Disabled {
		return
	}

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	// todo : replace strconv.FormatInt with strconv.AppendInt (see logger.go)

	// old logline
	// {"level":"info","id":30064773474,"status":200,
	// "method":"GET","path":"/","ip":"127.0.0.1","latency":0.118,
	// "user-agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
	// "time":"2025-09-05T09:16:19.07180518Z","caller":"router.go:120"}

	// new logline
	// {"level":"info","id":"4294967297","status":200,
	// "method":"GET","path":"/","ip":"127.0.0.1","latency_ms":0,
	// "user-agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
	// "time":"2025-09-05T10:21:50.609890725Z","caller":"fiber.go:124"}

	// a small dirty hack for escaping from zerolog.RawJSON()
	// since RawJSON is not validate JSON from arg, we can use it
	// for logging without JSON "tree" nesting
	buf.WriteByte('"')
	buf.WriteString(strconv.FormatInt(int64(c.Context().ID()), 10))
	buf.WriteByte('"')

	buf.WriteByte(',')
	buf.WriteString("\"status\":") // int
	buf.WriteString(strconv.FormatInt(int64(status), 10))

	buf.WriteByte(',')
	buf.WriteString("\"method\":") // string
	buf.WriteByte('"')
	buf.WriteString(c.Method())
	buf.WriteByte('"')

	buf.WriteByte(',')
	buf.WriteString("\"path\":") // string
	buf.WriteByte('"')
	buf.WriteString(c.Path())
	buf.WriteByte('"')

	buf.WriteByte(',')
	buf.WriteString("\"ip\":") // string
	buf.WriteByte('"')
	buf.WriteString(IPFromFiberRequest(c))
	buf.WriteByte('"')

	buf.WriteByte(',')
	buf.WriteString("\"latency_ms\":") // int
	buf.WriteString(strconv.FormatInt(lap.Milliseconds(), 10))

	buf.WriteByte(',')
	buf.WriteString("\"user-agent\":") // string
	buf.WriteByte('"')
	buf.WriteString(c.Get(fiber.HeaderUserAgent))
	buf.WriteByte('"')

	if msg != "" {
		buf.WriteByte(',')
		buf.WriteString("\"message\":") // string
		buf.WriteByte('"')
		buf.WriteString(msg)
		buf.WriteByte('"')
	}

	l.WithLevel(lvl).RawJSON("id", buf.B).Send() // int
}

// Mobile detector
var uaSecHeaderMobile = []byte("?1")

func IsRequestFromMobile(c *fiber.Ctx, ismob string) bool {
	var secua []byte
	if secua = c.Request().Header.Peek("Sec-CH-UA-Mobile"); len(secua) != 0 {
		return bytes.Equal(secua, uaSecHeaderMobile)
	}

	return len(c.Request().Header.Peek(ismob)) != 0
}

// 16 bytes for v4 with extra space
const IP_SIZE = 20

// IP store header
const IP_RESPONSE_HEADER = "x-client-ip"

// Dictionary for the fastest covertations from net.IP to string
// There is no allocations in high load setups (while ddos is processing)
//
// It's not idiomatic way of using Golang and it's like peace of shit
// but in 100k rps net.IP.String() spawning too much allocs...
var IPByteToString map[byte]string

func init() {
	IPByteToString = make(map[byte]string, 256)

	for n := range 256 {
		IPByteToString[byte(n)] = strconv.Itoa(n)
	}
}

type IP struct {
	b []byte
}

var ipPool = sync.Pool{New: func() any {
	return &IP{make([]byte, 0, IP_SIZE)}
}}

func IPFromFiberRequest(c *fiber.Ctx) string {
	// try to get IP from realip
	xip := c.Request().Header.Peek(c.App().Config().ProxyHeader)
	if len(xip) != 0 {
		return UnsafeString(xip)
	}

	// get IP from "cache"
	if hip := c.Response().Header.Peek(IP_RESPONSE_HEADER); len(hip) > 0 {
		return UnsafeString(hip)
	}

	// parse IP and cache it in Response headers
	ip := ipPool.Get().(*IP)
	ip.b = NetIP4ToBytes(ip.b, c.Context().RemoteIP())

	// * set the same key as ProxyTrusted for further resource economy
	// ! but from now, we need to delete this header after request processing
	c.Response().Header.AddBytesV(IP_RESPONSE_HEADER, ip.b)

	ip.b = ip.b[:0]
	ipPool.Put(ip)

	return UnsafeString(c.Response().Header.Peek(IP_RESPONSE_HEADER))
}

func NetIP4ToBytes(dst []byte, ip net.IP) []byte {
	if cap(dst) < IP_SIZE { // 255 . 255 . 255 . 255 (.)
		dst = append(dst, make([]byte, IP_SIZE-len(dst))...)
	}
	dst = dst[:0]

	for _, j := range ip {
		dst = append(dst, UnsafeBytes(IPByteToString[j])...)
		dst = append(dst, byte('.'))
	}

	return dst[:len(dst)-1]
}
