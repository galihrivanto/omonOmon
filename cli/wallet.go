package cli

import (
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
	Use:   "generate",
	Short: "Generate a new wallet",
	Run: func(cmd *cobra.Command, args []string) {
		wallet.GenerateWallet()
	},
}

var balanceCmd = &cobra.Command{
	Use:   "balance [address]",
	Short: "Check wallet balance",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wallet.CheckBalance(args[0])
	},
}

var sendCmd = &cobra.Command{
	Use:   "send [privateKey] [toAddress] [amount]",
	Short: "Send MON",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		amount, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			log.Fatal("Invalid amount")
		}
		wallet.SendMON(args[0], args[1], amount)
	},
}

func init() {
	WalletCmd.AddCommand(generateCmd)
	WalletCmd.AddCommand(balanceCmd)
	WalletCmd.AddCommand(sendCmd)
}
