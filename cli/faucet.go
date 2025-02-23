package cli

import (
	"fmt"
	"log"

	"github.com/galihrivanto/omonOmon/faucet"
	"github.com/spf13/cobra"
)

var FaucetCmd = &cobra.Command{
	Use:   "faucet [type] [address]",
	Short: "Claim MON from the faucet",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatal("faucet type and address are required")
		}

		faucetType := args[0]
		address := args[1]

		fmt.Println("Claiming MON from faucet", faucetType, "for address", address)

		if err := faucet.Claim(faucetType, address); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Claimed MON successfully")
	},
}
