package rsa

/*
#cgo pkg-config: openssl
#cgo CFLAGS: -O3 -Wall -Wno-deprecated-declarations
#include <openssl/evp.h>

EVP_PKEY* pkey_from_ne_u32(const unsigned char* n, size_t n_len,
                                unsigned long e_u32);
void free_pkey(EVP_PKEY* pkey);
int verify_pkcs1_v15_sha256_digest_pkey(
    EVP_PKEY* pkey,
    const unsigned char digest[32],
    const unsigned char* sig, size_t sig_len);
*/
import "C"
import (
	"unsafe"
)

type PubKey struct {
	p *C.EVP_PKEY
}

// fast method for e=65537:
func newPKeyFromNEU32(n []byte) *PubKey {
	if len(n) == 0 {
		return nil
	}

	pk := acquirePubKey()
	pk.p = C.pkey_from_ne_u32((*C.uchar)(unsafe.Pointer(&n[0])), C.size_t(len(n)),
		C.ulong(65537))
	if pk.p == nil {
		return nil
	}

	return pk
}

func (k *PubKey) Close() {
	if k == nil {
		return
	}

	if k.p != nil {
		C.free_pkey(k.p)
		k.p = nil
	}

	releasePubKey(k)
}

type RSAManager struct {
	ch *cache
}

func NewRSAManager(ccap int) *RSAManager {
	return &RSAManager{
		ch: newCache(ccap),
	}
}

func (m *RSAManager) VerifyDigestWithMod(n []byte, h [32]byte, dig, sig []byte) bool {
	if m == nil || m.ch == nil {
		panic("rsamanager is not initialized")
	}

	if len(dig) == 0 || len(sig) == 0 {
		return false
	}

	var en *entry
	if en = m.ch.getElement(h); en == nil {
		var pk *PubKey
		if pk = newPKeyFromNEU32(n); pk == nil {
			return false
		}

		en = m.ch.newElement(h, pk)
	}
	defer m.ch.release(en)

	return C.verify_pkcs1_v15_sha256_digest_pkey(
		en.pk.p,
		(*C.uchar)(unsafe.Pointer(&dig[0])),
		(*C.uchar)(unsafe.Pointer(&sig[0])), C.size_t(len(sig)),
	) == 1
}

func (m *RSAManager) Close() {
	m.ch.close()
}
