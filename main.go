package main

import (
	"fmt"
	"os"

	"github.com/galihrivanto/omonOmon/cli"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "omonOmon",
	Short: "A command-line wallet for Monad network",
}

func main() {
	rootCmd.AddCommand(cli.WalletCmd)
	rootCmd.AddCommand(cli.FaucetCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
