package utils

import (
	"strings"
	"sync"
)

var stringsBufferPool = sync.Pool{New: func() any {
	return &strings.Builder{}
}}

func AcquireStringsBuffer() *strings.Builder  { return stringsBufferPool.Get().(*strings.Builder) }
func ReleaseStringsBuffer(v *strings.Builder) { v.Reset(); stringsBufferPool.Put(v) }
