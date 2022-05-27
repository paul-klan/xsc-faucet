package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

type TxTokenBuild struct {
	client          *ethclient.Client
	privateKey      *ecdsa.PrivateKey
	signer          types.Signer
	fromAddress     common.Address
	contractAddress common.Address
	chainId         *big.Int
}

func NewTxTokenBuilder(provider, contractAddress string, privateKey *ecdsa.PrivateKey, chainId *big.Int) (*TxTokenBuild, error) {
	client, err := ethclient.Dial(provider)
	if err != nil {
		return nil, err
	}

	if chainId == nil {
		chainId, err = client.ChainID(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return &TxTokenBuild{
		client:          client,
		privateKey:      privateKey,
		signer:          types.NewEIP155Signer(chainId),
		fromAddress:     crypto.PubkeyToAddress(privateKey.PublicKey),
		contractAddress: common.HexToAddress(contractAddress),
	}, nil
}

func (b *TxTokenBuild) Sender() common.Address {
	return b.fromAddress
}

func (b *TxTokenBuild) Transfer(ctx context.Context, to string, amt *big.Int) (common.Hash, error) {
	log.Infof("ERC20TokenTranser contractAddress: %s, fromAddress: %s, toAddress: %s, amount: %s",
		b.contractAddress.Hex(), b.fromAddress.Hex(), to, amt.String())

	nonce, err := b.client.PendingNonceAt(ctx, b.fromAddress)
	if err != nil {
		return common.Hash{}, err
	}

	gasLimit := uint64(100000)
	gasPrice, err := b.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	toAddress := common.HexToAddress(to)
	value := big.NewInt(0)
	tokenAddress := b.contractAddress

	transferFnSignature := []byte("transfer(address,uint256)")
	// Get the method ID of the function
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4] // The first 4 bytes of the resulting hash is the methodId
	fmt.Printf("Method ID: %s\n", hexutil.Encode(methodID))

	// zero pad (to the left) the account address. The resulting byte slice must be 32 bytes long.
	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	fmt.Printf("To address: %s\n", hexutil.Encode(paddedAddress))

	amount := new(big.Int)
	amount.SetString("1000000000000000000", 10) // 1 token
	// zero pad (to the left) the amount. The resulting byte slice must be 32 bytes long.
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	fmt.Printf("Token amount: %s\n", hexutil.Encode(paddedAmount))

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	unsignedTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &tokenAddress,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	//unsignedTx := types.NewTransaction(nonce, b.contractAddress, value, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(unsignedTx, b.signer, b.privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	if err := b.client.SendTransaction(ctx, signedTx); err != nil {
		log.Errorf("ERC20TokenTranser SendTransaction error, %v", err)
		return common.Hash{}, err
	}

	log.Infof("ERC20TokenTranser contractAddress: %s, fromAddress: %s, toAddress: %s with amount %s",
		b.contractAddress.Hex(), b.fromAddress.Hex(), to, amt.String())

	log.Infof("txid: %s", signedTx.Hash().Hex())

	if _, err := bind.WaitMined(context.Background(), b.client, signedTx); err != nil {
		log.Errorf("mintToken WaitMined error, %v", err)
		return signedTx.Hash(), nil
	}

	return signedTx.Hash(), nil
}
