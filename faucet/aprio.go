package faucet

import (
	"errors"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type AprioFaucetClaimer struct{}

func (f *AprioFaucetClaimer) Claim(address string) error {
	if address == "" {
		return errors.New("address is required")
	}

	browser := rod.New()
	defer browser.Close()

	page, err := stealth.Page(browser)
	if err != nil {
		return err
	}

	fmt.Println("Navigating to faucet...")
	err = page.Navigate("https://stake.apr.io/faucet")
	if err != nil {
		return err
	}

	button, err := page.Element("main button")
	if err != nil {
		return err
	}

	return button.Click(proto.InputMouseButtonLeft, 1)
}

func init() {
	factories["aprio"] = func() FaucetClaimer { return &AprioFaucetClaimer{} }
}
