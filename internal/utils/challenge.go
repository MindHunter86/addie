package utils

import "math"

// calculated for RSA8192 usage buffer size
// urlRawB64(modulus == signature == buffsize)
//
// RSA2048 - about 344 bytes
// * RSA4096 - about 684 bytes
// RSA8192 - about 1368 bytes
// total : base64url doubled modulus size for b64 decoding
// and + 20% as extra space
var SANITIZED_MODULUS_BUFSIZE = int(684*2 + math.Floor(684*0.2))

// payload size ~128 bytes (2025/07)
// so we use doubled buffer for further unbase64ing
// and 20% from size as extra space
var SANITIZED_PAYLOAD_SIZE = int(128*2 + math.Floor(128*0.2))
