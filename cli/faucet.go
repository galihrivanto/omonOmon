package cli

import (
	"fmt"
	"log"

	"github.com/galihrivanto/omonOmon/faucet"
	"github.com/spf13/cobra"
)

var FaucetCmd = &cobra.Command{
	Use:   "faucet [address]",
	Short: "Claim MON from the faucet",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("address is required")
		}

		if err := faucet.Claim(args[0]); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Claimed MON successfully")
	},
}
