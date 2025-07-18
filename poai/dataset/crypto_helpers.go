package dataset

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"

	"golang.org/x/crypto/sha3"
)

func verifySHA256(rec, want []byte) bool {
	got := sha3.Sum256(rec)
	return bytes.Equal(got[:], want)
}

func aesgcmDecrypt(key, cipherText, nonce []byte) ([]byte, error) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}
