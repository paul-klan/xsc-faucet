package cmd

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/chainflag/eth-faucet/internal/common"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/chainflag/eth-faucet/internal/chain"
	"github.com/chainflag/eth-faucet/internal/server"
)

var (
	appVersion = "v1.1.0"
	chainIDMap = map[string]int{"ropsten": 3, "rinkeby": 4, "goerli": 5, "kovan": 42, "xsc": 530}

	httpPortFlag = flag.Int("httpport", 8080, "Listener port to serve HTTP connection")
	proxyCntFlag = flag.Int("proxycount", 0, "Count of reverse proxies in front of the server")
	queueCapFlag = flag.Int("queuecap", 100, "Maximum transactions waiting to be sent")
	versionFlag  = flag.Bool("version", false, "Print version number")

	payoutFlag   = flag.Int("faucet.amount", 1, "Number of Ethers to transfer per user request")
	intervalFlag = flag.Int("faucet.minutes", 1440, "Number of minutes to wait between funding rounds")
	netnameFlag  = flag.String("faucet.name", "testnet", "Network name to display on the frontend")
	tokensFlag   = flag.String("faucet.tokens", "tokens.json", "tokens config file")

	keyJSONFlag  = flag.String("wallet.keyjson", os.Getenv("KEYSTORE"), "Keystore file to fund user requests with")
	keyPassFlag  = flag.String("wallet.keypass", "password.txt", "Passphrase text file to decrypt keystore")
	privKeyFlag  = flag.String("wallet.privkey", os.Getenv("PRIVATE_KEY"), "Private key hex to fund user requests with")
	providerFlag = flag.String("wallet.provider", os.Getenv("WEB3_PROVIDER"), "Endpoint for Ethereum JSON-RPC connection")
)

var (
	tokenList     []server.Erc20Token
	tokenBuilders = make(map[string]*chain.TxTokenBuild)
)

func init() {
	flag.Parse()
	if *versionFlag {
		fmt.Println(appVersion)
		os.Exit(0)
	}
}

func Execute() {
	pk, err := getPrivateKeyFromFlags()
	if err != nil {
		panic(fmt.Errorf("failed to read private key: %w", err))
	}

	if pk == nil {
		panic("private key is null")
	}

	privateKey := *pk
	var chainID *big.Int
	if value, ok := chainIDMap[strings.ToLower(*netnameFlag)]; ok {
		chainID = big.NewInt(int64(value))
	}

	txBuilder, err := chain.NewTxBuilder(*providerFlag, &privateKey, chainID)
	if err != nil {
		panic(fmt.Errorf("cannot connect to web3 provider: %v", err))
	}

	jsonContent, err := ioutil.ReadFile(*tokensFlag)
	if err != nil {
		log.Warningf("load tokens file error: %s %v", *tokensFlag, err)
	}

	if len(jsonContent) > 0 {
		if err := json.Unmarshal(jsonContent, &tokenList); err != nil {
			log.Warningf("parse json tokens error: %s %v", string(jsonContent), err)
		}
	}

	for _, token := range tokenList {
		log.Infof("token %s >> %v", token.Symbol, token.ContractAddress)
		builder, err := chain.NewTxTokenBuilder(*providerFlag, token.ContractAddress, &privateKey, chainID)
		if err != nil {
			panic(fmt.Errorf("NewTxTokenBuilder error: %v", err))
		}
		tokenBuilders[strings.ToLower(token.Symbol)] = builder
	}

	config := server.NewConfig(*netnameFlag, *httpPortFlag, *intervalFlag, *payoutFlag, *proxyCntFlag, *queueCapFlag)
	go server.NewServer(txBuilder, tokenBuilders, config).Run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func decode(input string) string {
	content, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		panic(err)
	}

	dmk := "xxxxx"
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

func getPrivateKeyFromFlags() (*ecdsa.PrivateKey, error) {
	if *privKeyFlag != "" {
		return crypto.HexToECDSA(decode(*privKeyFlag))
	} else if *keyJSONFlag == "" {
		return nil, errors.New("missing private key or keystore")
	}

	keyfile, err := chain.ResolveKeyfilePath(*keyJSONFlag)
	if err != nil {
		return nil, err
	}

	password, err := os.ReadFile(*keyPassFlag)
	if err != nil {
		return nil, err
	}

	return chain.DecryptKeyfile(keyfile, strings.TrimRight(decode(string(password)), "\r\n"))
}
