//go:build go1.20 && !windows && !plan9

package syslog

import (
	"unsafe"
)

// UnsafeString returns a string pointer without allocation
func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b)) // skipcq: GSC-G103 it's ok here
}

// UnsafeBytes returns a byte pointer without allocation.
func unsafeBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s)) // skipcq: GSC-G103 it's ok here
}
