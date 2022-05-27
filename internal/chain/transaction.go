package chain

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"math/big"
)

type TxBuilder interface {
	Sender() common.Address
	Transfer(ctx context.Context, to string, value *big.Int) (common.Hash, error)
}

type TxBuild struct {
	client      *ethclient.Client
	privateKey  *ecdsa.PrivateKey
	signer      types.Signer
	fromAddress common.Address
}

func NewTxBuilder(provider string, privateKey *ecdsa.PrivateKey, chainID *big.Int) (TxBuilder, error) {
	client, err := ethclient.Dial(provider)
	if err != nil {
		return nil, err
	}

	if chainID == nil {
		chainID, err = client.ChainID(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return &TxBuild{
		client:      client,
		privateKey:  privateKey,
		signer:      types.NewEIP155Signer(chainID),
		fromAddress: crypto.PubkeyToAddress(privateKey.PublicKey),
	}, nil
}

func (b *TxBuild) Sender() common.Address {
	return b.fromAddress
}

func (b *TxBuild) Transfer(ctx context.Context, to string, value *big.Int) (common.Hash, error) {
	log.Infof("transer >> contractAddress: fromAddress: %s toAddress: %s  amount:  %s",
		b.fromAddress.Hex(), to, value.String())
	nonce, err := b.client.PendingNonceAt(ctx, b.Sender())
	if err != nil {
		return common.Hash{}, err
	}

	gasLimit := uint64(1000000)
	gasPrice, err := b.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	toAddress := common.HexToAddress(to)
	unsignedTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
	})

	signedTx, err := types.SignTx(unsignedTx, b.signer, b.privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	if err := b.client.SendTransaction(ctx, signedTx); err != nil {
		log.Errorf("builder SendTransaction error, %v", err)
		return common.Hash{}, err
	}

	log.Infof("transer contractAddress: fromAddress: %s, toAddress: %s with amount %s",
		b.fromAddress.Hex(), to, value.String())

	log.Infof("tx-hash: %s", signedTx.Hash().Hex())

	if _, err := bind.WaitMined(context.Background(), b.client, signedTx); err != nil {
		log.Errorf("mintToken WaitMined error, %v", err)
		return signedTx.Hash(), nil
	}

	return signedTx.Hash(), nil
}
