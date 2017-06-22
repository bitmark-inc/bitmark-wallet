package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func readMnemonic() (string, error) {
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return "", err
	}

	tmpIO, err := os.OpenFile("/dev/tty", os.O_RDWR, os.ModePerm)
	if err != nil {
		return "", err
	}
	console := terminal.NewTerminal(tmpIO, "")
	defer terminal.Restore(0, oldState)
	console.Write([]byte("IMPORTANT: all the data in the existance wallet will be removed.\n"))
	console.SetPrompt("Enter the mnemonic phrases for a wallet: ")
	return console.ReadLine()
}

func readPassword(prompt string, passLen int) (string, error) {

	password := os.Getenv("WALLET_PASSWORD")
	if password != "" {
		if len(password) > passLen {
			return password, nil
		} else {
			fmt.Println("Invalid environment: WALLET_PASSWORD")
		}
	}

	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return "", err
	}

	tmpIO, err := os.OpenFile("/dev/tty", os.O_RDWR, os.ModePerm)
	if err != nil {
		return "", err
	}
	passwordConsole := terminal.NewTerminal(tmpIO, "")
	password, err = passwordConsole.ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	terminal.Restore(0, oldState)

	if passLen > 0 && len(password) < passLen {
		return "", fmt.Errorf("password length less than 8")
	}

	return password, nil
}
