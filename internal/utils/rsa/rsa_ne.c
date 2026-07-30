// rsa_ne.c
#include <openssl/evp.h>
#include <openssl/rsa.h>
#include <openssl/err.h>
#include <string.h>

// build instructions:
// cc -O3 -fPIC -c rsa_ne.c `pkg-config --cflags openssl`
// cc -shared -o librsa_ne.so rsa_ne.o `pkg-config --libs openssl`

// the fastest method for 32bit E
EVP_PKEY *pkey_from_ne_u32(const unsigned char *n, size_t n_len,
                           unsigned long e_u32)
{
    if (!n || n_len == 0 || e_u32 == 0)
        return NULL;

    EVP_PKEY *pkey = NULL;
    RSA *rsa = NULL;
    BIGNUM *bn_n = NULL, *bn_e = NULL;

    bn_n = BN_bin2bn(n, (int)n_len, NULL);
    bn_e = BN_new();
    if (!bn_n || !bn_e)
        goto err;
    if (BN_set_word(bn_e, e_u32) != 1)
        goto err;

    rsa = RSA_new();
    if (!rsa)
        goto err;

    if (RSA_set0_key(rsa, bn_n, bn_e, NULL) != 1)
        goto err;
    bn_n = NULL;
    bn_e = NULL;

    pkey = EVP_PKEY_new();
    if (!pkey)
        goto err;
    if (EVP_PKEY_assign_RSA(pkey, rsa) != 1)
        goto err;
    rsa = NULL;
    return pkey;

err:
    if (bn_n)
        BN_free(bn_n);
    if (bn_e)
        BN_free(bn_e);
    if (rsa)
        RSA_free(rsa);
    if (pkey)
        EVP_PKEY_free(pkey);
    return NULL;
}

// release resources
void free_pkey(EVP_PKEY *pkey)
{
    if (pkey)
        EVP_PKEY_free(pkey);
}

// 32 byte digits (sha256) verification
int verify_pkcs1_v15_sha256_digest_pkey(
    EVP_PKEY *pkey,
    const unsigned char digest[32],
    const unsigned char *sig, size_t sig_len)
{
    if (!pkey || !digest || !sig)
        return 0;

    RSA *rsa = EVP_PKEY_get0_RSA(pkey);
    if (!rsa)
        return 0;

    // fast check for length
    // TODO : maybe delete it
    if ((int)sig_len != RSA_size(rsa))
        return 0;

    int ok = RSA_verify(NID_sha256, digest, 32, sig, (unsigned int)sig_len, rsa);
    return ok == 1 ? 1 : 0;
}
