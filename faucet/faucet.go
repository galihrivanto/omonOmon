package faucet

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

func Claim(address string) error {
	if address == "" {
		return errors.New("address is required")
	}

	browser := rod.New()
	defer browser.Close()

	fmt.Println("Connecting...")
	if err := browser.Connect(); err != nil {
		return err
	}

	// create stealth page
	page, err := stealth.Page(browser)
	if err != nil {
		return err
	}

	fmt.Println("Navigating to faucet...")
	err = page.Navigate("https://testnet.monad.xyz/")
	if err != nil {
		return err
	}

	element, err := page.Element(".wallet-address-container input")
	if err != nil {
		return err
	}

	button, err := page.Element("button")
	if err != nil {
		return err
	}

	fmt.Println("Entering address...")
	err = element.Input(address)
	if err != nil {
		return err
	}

	fmt.Println("Clicking button...")
	err = button.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return err
	}

	return page.WaitIdle(30 * time.Second)
}
