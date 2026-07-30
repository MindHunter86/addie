package rsa

import (
	"sync"
)

var pubKeyPool = sync.Pool{New: func() any {
	return &PubKey{}
}}

func acquirePubKey() *PubKey { return pubKeyPool.Get().(*PubKey) }
func releasePubKey(v *PubKey) {
	pubKeyPool.Put(v)
}
