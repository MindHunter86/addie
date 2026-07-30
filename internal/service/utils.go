package service

import (
	"io"

	"github.com/MindHunter86/addie/internal/stats"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/valyala/bytebufferpool"

	"github.com/gofiber/fiber/v2"
	futils "github.com/gofiber/fiber/v2/utils"
)

// The fastest way to generate JSON challenge for failed traffic
//
// * DISCLAIMER
// Since we have too much allocs in JSON marshaller, here we have a
// hardcoded JSON generation. Due to a big amount of trafic (DDOS inc.)
// I think it's not a huge problem, to use hardcoding here
func writeJsonErrorFastTo(to io.Writer, c int, m string) (e error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	buf.WriteByte('{')

	buf.WriteString("\"error\":")
	buf.WriteByte('"')
	// buf.WriteString(strconv.Itoa(c)) - allocs here
	buf.WriteString(futils.StatusMessage(c))
	buf.WriteByte('"')

	buf.WriteByte(',')
	buf.WriteString("\"message\":")
	buf.WriteByte('"')
	buf.Write(utils.UnsafeBytes(m))
	buf.WriteByte('"')

	buf.WriteByte(',')
	buf.WriteString("\"detail\":")
	buf.WriteByte('"')
	buf.WriteString("unavailable in production mode")
	buf.WriteByte('"')

	buf.WriteByte('}')

	_, e = buf.WriteTo(to)
	return
}

// fiber 404 handler with minimal allocations
func fiber404ErrorHandler(*fiber.Ctx) error {
	return utils.AcquireFiberError(fiber.StatusNotFound, "requested page could not be found")
}

func statIncMetricFiberHandler(m stats.IncrementMetric) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		utils.ContextValueExtract[*stats.Stats](ctx, utils.CtxStats).WriteIncMetric(m)
		return c.Next()
	}
}
