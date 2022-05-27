package common

import (
	"crypto/md5"
	"encoding/hex"
)

func GenMd5(input []byte) []byte {
	m := md5.New()
	m.Write(input)
	return m.Sum(nil)
}

func GenMd5WithHex(input []byte) string {
	m := md5.New()
	m.Write(input)
	return hex.EncodeToString(m.Sum(nil))
}
