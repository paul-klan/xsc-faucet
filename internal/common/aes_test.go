package common

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

const body = `{
    "namet783t2837y3824843h4888",
}`

var (
	defaultKey = "12345678901234561234567890123456"
	defaultIv  = "1234567890123456"
)

func TestAESCBC(t *testing.T) {

	encodeData, _ := AesEncryptCBC([]byte(body), []byte(defaultKey), []byte(defaultIv))
	t.Logf("encode [%s]", hex.EncodeToString(encodeData))
	t.Logf("base64 [%s]", base64.StdEncoding.EncodeToString(encodeData))

	rawData, _ := AesDecryptCBC(encodeData, []byte(defaultKey), []byte(defaultIv))
	t.Logf("decode [%s]", rawData)
	t.Logf("base64 [%s]", base64.StdEncoding.EncodeToString(rawData))

	t.Logf("[%s]", GenMd5WithHex([]byte("t7882383443")))
}
