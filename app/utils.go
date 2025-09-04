package app

import (
	"encoding/base64"
	"slices"

	"github.com/gofiber/utils"
)

func xorEncryptDecrypt(k, p []byte) []byte {
	kl := len(k)

	for i := range len(p) {
		p[i] = p[i] ^ k[i%kl]
	}

	return p
}

func copyPayloadToBuffer(dst, src []byte) []byte {
	if cap(dst) < len(src) {
		dst = append(dst, make([]byte, len(src)-len(dst))...)
	}
	dst = dst[:len(src)]

	copy(dst, src)
	return dst
}

func extractBase64Value(dst, src []byte) (_ []byte, e error) {
	dst = copyPayloadToBuffer(dst, src)

	ps, raws := len(dst), base64.RawURLEncoding.DecodedLen(len(dst))

	// make reverse for further reverse after appending slice to the begining
	slices.Reverse(dst)

	if cap(dst) < ps+raws {
		dst = append(dst, make([]byte, (ps+raws)-ps)...)
	}
	dst = dst[:ps+raws]

	// move raw data to the end of slice
	slices.Reverse(dst)

	// decode
	if _, e = base64.RawURLEncoding.Decode(dst[:raws], dst[raws:ps+raws]); e != nil {
		return
	}

	// cut garbage (raw data)
	dst = dst[:raws]
	return dst, e
}

func B64(src []byte) string {
	var dst []byte

	b64l, b64sz := len(dst), base64.RawURLEncoding.EncodedLen(len(src))

	if cap(dst) < b64sz {
		dst = append(dst, make([]byte, b64sz-b64l)...)
	}
	dst = dst[:b64sz]

	base64.RawURLEncoding.Encode(dst, src)

	// add security bytes for random data protection
	// dst = m.SetSecurityBytes(dst)

	// respond with temporary b64 slice
	return utils.UnsafeString(dst)
}
