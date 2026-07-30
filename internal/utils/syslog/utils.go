//go:build !windows && !plan9

package syslog

import "sync"

const DefaultMessageBufferSize = 768
const DefaultMessageTimeBufSize = 64

var syslogMsgPool = sync.Pool{New: func() any {
	return &syslogMsg{
		b:     make([]byte, 0, DefaultMessageBufferSize),
		timeb: make([]byte, 0, DefaultMessageTimeBufSize),
	}
}}

func acquireSyslogMsg() *syslogMsg { return syslogMsgPool.Get().(*syslogMsg) }
func releaseSyslogMsg(v *syslogMsg) {
	v.release()
	syslogMsgPool.Put(v)
}
