package wallet_test

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/bitmark-inc/bitmark-wallet"
)

func Example_newWallet() {
	seedHex := "fded5e8970380eef15f742348d28511111366ae6a55188402b16c69922006fe6"
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		log.Fatal(err)
	}
	walletData := "wallet.dat"
	w := wallet.New(seed, walletData)
	coinAccount, err := w.CoinAccount(wallet.BTC, wallet.Test(true), 0)
	if err != nil {
		log.Fatal(err)
	}
	address, err := coinAccount.NewExternalAddr()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(address)
	// Output: mjWfeKTxhPD2eA9FbVrqTkGtZD19fHEgaU
}
