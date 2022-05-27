// this code from https://github.com/wumansgy/goEncrypt/blob/master/aescbc.go

package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

var (
	ErrKeyLengthSixteen = errors.New("invalid aes key")
	ErrIvLengthSixteen  = errors.New("invalid aes iv")
)

// encrypt
func AesEncryptCBC(plainText, key, ivAes []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrKeyLengthSixteen
	}

	if len(ivAes) != 16 {
		return nil, ErrIvLengthSixteen
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	paddingText := PKCS5Padding(plainText, block.BlockSize())

	blockMode := cipher.NewCBCEncrypter(block, ivAes)
	cipherText := make([]byte, len(paddingText))
	blockMode.CryptBlocks(cipherText, paddingText)

	return cipherText, nil
}

// decrypt
func AesDecryptCBC(cipherText, key, ivAes []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrKeyLengthSixteen
	}

	if len(ivAes) != 16 {
		return nil, ErrIvLengthSixteen
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockMode := cipher.NewCBCDecrypter(block, ivAes)
	paddingText := make([]byte, len(cipherText))
	blockMode.CryptBlocks(paddingText, cipherText)

	plainText := PKCS5UnPadding(paddingText)
	return plainText, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
