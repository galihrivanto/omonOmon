package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	RPC_URL  = "https://testnet-rpc.monad.xyz"
	CHAIN_ID = 210425
)

type Wallet struct {
	Address    string
	PrivateKey string
}

// Generate a new wallet
func GenerateWallet() *Wallet {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	privateKeyHex := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	return &Wallet{
		Address:    address,
		PrivateKey: privateKeyHex,
	}
}

// Load a wallet from a private key
func LoadWallet(path string) *Wallet {
	privateKeyHex, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA(string(privateKeyHex))
	if err != nil {
		log.Fatal(err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	return &Wallet{
		Address:    address,
		PrivateKey: string(privateKeyHex),
	}
}

// Check balance
func (w *Wallet) Balance() (*big.Float, error) {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		return nil, err
	}

	account := common.HexToAddress(w.Address)
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		return nil, err
	}

	return new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18)), nil
}

// Send send tokens (MON) to an address
func (w *Wallet) Send(toAddress string, amount float64) (string, error) {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		return "", err
	}

	privateKey, err := crypto.HexToECDSA(w.PrivateKey)
	if err != nil {
		return "", err
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", err
	}

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	value := new(big.Int)
	value.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)

	to := common.HexToAddress(toAddress)
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, nil)

	chainID := big.NewInt(CHAIN_ID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

// Save a wallet to a file
// currently only naive implementation
func (w *Wallet) Save(path string) error {
	if err := os.WriteFile(path, []byte(w.PrivateKey), 0644); err != nil {
		return err
	}

	fmt.Println("Wallet saved to", path)
	return nil
}
