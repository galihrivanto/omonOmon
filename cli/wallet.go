package cli

import (
	"fmt"
	"log"
	"strconv"

	"github.com/galihrivanto/omonOmon/wallet"
	"github.com/spf13/cobra"
)

var WalletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Manage Monad wallets",
}

var generateCmd = &cobra.Command{
	Use:   "generate [walletPath]",
	Short: "Generate a new wallet",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w := wallet.GenerateWallet()
		fmt.Println("Address:", w.Address)
		fmt.Println("Private Key:", w.PrivateKey)

		// save wallet to file
		if err := w.Save(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Check wallet balance",
	Run: func(cmd *cobra.Command, args []string) {
		walletPath, _ := cmd.Flags().GetString("wallet-path")

		w := wallet.LoadWallet(walletPath)
		balance, err := w.Balance()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Balance:", balance, "MON")
	},
}

var sendCmd = &cobra.Command{
	Use:   "send [toAddress] [amount]",
	Short: "Send MON",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		walletPath, _ := cmd.Flags().GetString("wallet-path")

		amount, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			log.Fatal("Invalid amount")
		}
		w := wallet.LoadWallet(walletPath)

		txHash, err := w.Send(args[0], amount)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Transaction Hash:", txHash)
	},
}

var walletConnectCmd = &cobra.Command{
	Use:   "wallet-connect [walletConnectURI]",
	Short: "Connect to a wallet using WalletConnect",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		walletPath, _ := cmd.Flags().GetString("wallet-path")

		w := wallet.LoadWallet(walletPath)
		if err := w.WalletConnect(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	WalletCmd.PersistentFlags().StringP("wallet-path", "w", ".wallet", "Wallet path")
	WalletCmd.AddCommand(generateCmd)
	WalletCmd.AddCommand(balanceCmd)
	WalletCmd.AddCommand(sendCmd)
	WalletCmd.AddCommand(walletConnectCmd)
}
