//go:build !windows && !plan9

package syslog

import (
	"io"
	"strconv"
	"time"
)

type syslogMsg struct {
	b     []byte
	timeb []byte // 4 byte slice for integer appends
}

func (m *syslogMsg) release() {
	m.b = m.b[:0]
	m.timeb = m.timeb[:0]
}

func (m *syslogMsg) appendTime(f string) []byte {
	cmax := len(f) + 10 // from time/format.go:Format()

	if cap(m.timeb) < cmax {
		m.timeb = append(m.timeb, make([]byte, cmax-cap(m.timeb))...)
	}
	m.timeb = m.timeb[:0]

	m.timeb = time.Now().AppendFormat(m.timeb, f)
	return m.append(m.timeb)
}

func (m *syslogMsg) appendInt(p int) []byte {
	return m.appendString(strconv.FormatInt(int64(p), 10))
}

func (m *syslogMsg) appendString(p string) []byte {
	return m.append(unsafeBytes(p))
}

func (m *syslogMsg) append(p []byte) []byte {
	bl, sz := len(m.b), len(m.b)+len(p)

	if cap(m.b) < sz {
		m.b = append(m.b, make([]byte, sz-bl)...)
	}
	m.b = m.b[:bl]

	m.b = append(m.b, p...)
	return m.b
}

func (m *syslogMsg) writeTo(w io.Writer) (int64, error) {
	n, e := w.Write(m.b)
	return int64(n), e
}
