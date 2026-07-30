package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/urfave/cli/v2"
)

var syslogOverload sync.RWMutex

type levelWriter struct {
	io.Writer
	Level zerolog.Level
}

func (m *levelWriter) WriteLevel(l zerolog.Level, p []byte) (n int, err error) {
	if l >= m.Level { // Notice that it's ">=", not ">"
		return m.Writer.Write(p)
	}
	return len(p), nil
}

type Logger struct {
	SystemMultiLogger zerolog.Logger
	AccessMultiLogger zerolog.Logger

	diostd, diosys *diode.Writer
}

func NewLogger(c *cli.Context) (m *Logger, e error) {
	m = &Logger{}

	// global logger setup
	zerolog.CallerMarshalFunc = callerMarshalFunc
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// common non-blocking writer for stdout (max - 1k in sec)
	if m.diostd, e = m.stdoutWriterSetup(100, 100*time.Millisecond); e != nil {
		return nil, ExtraErrorWrapper(e, "bootstrap stdout writer")
	}

	// common non-blocking writer for syslog (max - 1k in sec)
	if m.diosys, e = m.syslogWriterSetup(c, 1000, 100*time.Millisecond); e != nil {
		return nil, ExtraErrorWrapper(e, "bootstrap syslog writer")
	}

	// bootstrap loggers
	if e = m.newSystemLogger(c); e != nil {
		return nil, ExtraErrorWrapper(e, "bootstrap system logger")
	}

	if e = m.newAccessLogger(c); e != nil {
		return nil, ExtraErrorWrapper(e, "bootstrap access logger")
	}

	return
}

func (m *Logger) SystemLogger() *zerolog.Logger {
	return &m.SystemMultiLogger
}

func (m *Logger) AccessLogger() *zerolog.Logger {
	return &m.AccessMultiLogger
}

func (m *Logger) Destroy() error {
	var errs []error
	if err := m.diostd.Close(); err != nil {
		errs = append(errs, ExtraErrorWrapper(err, "closing stdout diode buffer"))
	}

	if m.diosys != nil {
		if err := m.diosys.Close(); err != nil {
			errs = append(errs, ExtraErrorWrapper(err, "closing syslog diode buffer"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (m *Logger) newSystemLogger(c *cli.Context) (e error) {
	var lvl zerolog.Level
	if lvl, e = zerolog.ParseLevel(c.String("log-level")); e != nil {
		return ExtraErrorWrapper(e, "parsing 'log-level' log level")
	}
	zerolog.SetGlobalLevel(lvl)

	if m.diosys != nil {
		m.SystemMultiLogger = zerolog.New(zerolog.MultiLevelWriter(
			zerolog.ConsoleWriter{Out: m.diostd},
			m.diosys)).With().Timestamp().Caller().Logger().Level(lvl)
	} else {
		m.SystemMultiLogger = zerolog.New(zerolog.ConsoleWriter{Out: m.diostd}).
			With().Timestamp().Caller().Logger().Level(lvl)
	}

	return
}

func (m *Logger) newAccessLogger(c *cli.Context) (e error) {
	var stdlvl zerolog.Level
	if stdlvl, e = zerolog.ParseLevel(c.String("accesslog-level")); e != nil {
		return ExtraErrorWrapper(e, "parsing 'accesslog-level' log level")
	}

	var syslvl zerolog.Level
	if syslvl, e = zerolog.ParseLevel(c.String("accesslog-syslog-level")); e != nil {
		return ExtraErrorWrapper(e, "parsing 'accesslog-syslog-level' log level")
	}

	if stdlvl == zerolog.NoLevel {
		stdlvl = zerolog.Disabled
	}
	if syslvl == zerolog.NoLevel {
		syslvl = zerolog.Disabled
	}

	if stdlvl != zerolog.Disabled && syslvl != zerolog.Disabled && m.diosys != nil {
		m.AccessMultiLogger = zerolog.New(zerolog.MultiLevelWriter(
			&levelWriter{zerolog.ConsoleWriter{Out: m.diostd}, stdlvl},
			&levelWriter{m.diosys, syslvl})).With().Timestamp().Caller().Logger()
	} else if stdlvl != zerolog.Disabled {
		m.AccessMultiLogger = zerolog.New(zerolog.ConsoleWriter{Out: m.diostd}).
			With().Timestamp().Caller().Logger().Level(stdlvl)
	} else if syslvl != zerolog.Disabled && m.diosys != nil {
		m.AccessMultiLogger = zerolog.New(m.diosys).
			With().Timestamp().Caller().Logger().Level(syslvl)
	} else {
		m.AccessMultiLogger = zerolog.New(io.Discard).Level(zerolog.Disabled)
	}

	return
}

func (m *Logger) stdoutWriterSetup(msglmt int, lmtdur time.Duration) (_ *diode.Writer, _ error) {
	return m.syslogWriterSetup(nil, msglmt, lmtdur)
}

func (*Logger) syslogWriterSetup(c *cli.Context, msglmt int, lmtdur time.Duration) (_ *diode.Writer, e error) {
	if c == nil {
		dio := diode.NewWriter(os.Stdout, msglmt, lmtdur,
			dioAlerter("ACHTUNG! diode writer drops %d stdout messages\n", false))
		return &dio, nil
	}

	if c.String("syslog-server") == "" {
		return
	}

	if runtime.GOOS == "windows" {
		return nil, errors.New("syslog is not available for windows; golang doesn't support syslog for win systems")
	}

	var syslogio io.Writer
	if syslogio, e = setUpSyslogWriter(c); e != nil {
		return nil, ExtraErrorWrapper(e, "connecting to syslog server")
	}

	// common non-blocking writer for syslog (max - 5k in sec)
	dio := diode.NewWriter(syslogio, msglmt, lmtdur,
		dioAlerter("ACHTUNG! diode writer drops %d syslog messages\n", true))
	return &dio, nil
}

func dioAlerter(alrt string, syslog bool) func(int) {
	return func(i int) {
		fmt.Fprintf(os.Stderr, alrt, i)

		if syslog {
			go syslogBlockWriting()
		}
	}
}

func syslogBlockWriting() {
	fmt.Fprintln(os.Stderr, "blocking syslog output for 10 seconds")

	syslogOverload.Lock()
	tick := time.NewTicker(10 * time.Second)

	<-tick.C
	tick.Stop()
	syslogOverload.Unlock()
	fmt.Fprintln(os.Stderr, "syslog output unblocked")
}

func callerMarshalFunc(_ uintptr, file string, line int) string {
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}

	s := acquireStringsBuilder()
	defer releaseStringsBuilder(s)

	b := acquireBytesBuffer()
	defer releaseBytesBuffer(b)

	b.b = strconv.AppendInt(b.b, int64(line), 10)

	s.WriteString(short)
	s.WriteByte(':')
	s.WriteString(UnsafeString(b.b))

	file = s.String()
	return file
}

// todo : replace to pool.go and use in project too
var stringsBuilderPool = sync.Pool{New: func() any {
	return &strings.Builder{}
}}

func acquireStringsBuilder() *strings.Builder { return stringsBuilderPool.Get().(*strings.Builder) }
func releaseStringsBuilder(v *strings.Builder) {
	v.Reset()
	stringsBuilderPool.Put(v)
}

type bytesBuffer struct{ b []byte }

func (m *bytesBuffer) reset() {
	m.b = m.b[:0]
}

var bytesBufferPool = sync.Pool{New: func() any {
	return &bytesBuffer{
		b: make([]byte, 0, 8),
	}
}}

func acquireBytesBuffer() *bytesBuffer { return bytesBufferPool.Get().(*bytesBuffer) }
func releaseBytesBuffer(v *bytesBuffer) {
	v.reset()
	bytesBufferPool.Put(v)
}
