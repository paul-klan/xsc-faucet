package main

import (
	"encoding/base64"
	"fmt"
	"github.com/chainflag/eth-faucet/internal/common"
	"os"
)

var (
	dmk = "xxx"
)

func decode(input string) string {
	content, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		panic(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic("host name error," + err.Error())
	}
	div := common.GenMd5WithHex([]byte(hostname))[:16]

	raw, err := common.AesDecryptCBC(content, common.GenMd5([]byte(dmk)), []byte(div))
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func encode(input string) string {
	hostname, err := os.Hostname()
	if err != nil {
		panic("host name error," + err.Error())
	}
	div := common.GenMd5WithHex([]byte(hostname))[:16]

	raw, err := common.AesEncryptCBC([]byte(input), common.GenMd5([]byte(dmk)), []byte(div))
	if err != nil {
		panic(err)
	}

	content := base64.StdEncoding.EncodeToString(raw)
	return content
}

func main() {

	code := "sss"
	out := encode(code)

	fmt.Println(out)

	raw := decode(out)

	if raw == code {
		fmt.Println("success")
	} else {
		fmt.Println("failure")
	}
}
