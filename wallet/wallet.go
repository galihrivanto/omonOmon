package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const RPC_URL = "https://testnet-rpc.monad.xyz"
const CHAIN_ID = 210425

// Generate a new wallet
func GenerateWallet() {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	privateKeyHex := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	fmt.Println("Address:", address)
	fmt.Println("Private Key:", privateKeyHex)
}

// Check balance
func CheckBalance(address string) {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatal(err)
	}

	account := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Balance:", new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18)), "MON")
}

// Send MON
func SendMON(privateKeyHex, toAddress string, amount float64) {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	value := new(big.Int)
	value.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)

	to := common.HexToAddress(toAddress)
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, nil)

	chainID := big.NewInt(CHAIN_ID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction Sent! Hash:", signedTx.Hash().Hex())
}
