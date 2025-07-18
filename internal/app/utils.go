package app

func xorEncryptDescrypt(k string, p []byte) []byte {
	kl := len(k)

	for i := range len(p) {
		p[i] = p[i] ^ k[i%kl]
	}

	return p
}
