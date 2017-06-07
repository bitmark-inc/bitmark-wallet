package main

import (
	"bytes"
	"fmt"
	"path"
	"strconv"

	"encoding/hex"
	"github.com/bitmark-inc/bitmark-wallet"
	"github.com/bitmark-inc/bitmark-wallet/discover"
	"github.com/spf13/cobra"
)

var w *wallet.Wallet
var coinAccount *wallet.CoinAccount

var ltcCmd = &cobra.Command{
	Use:   "ltc",
	Short: "Litecoin wallet",
	Long:  `Litecoin wallet`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// PreRun only for subcommand
		if cmd.Use == "ltc" {
			return
		}

		datadir, err := cmd.Root().PersistentFlags().GetString("datadir")
		returnIfErr(err)

		conf, err := cmd.Root().PersistentFlags().GetString("conf")
		returnIfErr(err)

		dataFile := path.Join(datadir, conf)
		encryptedSeed, err := getConfig(dataFile, []byte("SEED"))
		returnIfErr(err)

		password, err := readPassword("Input wallet password: ", 0)
		returnIfErr(err)

		passHash := dblSHA256([]byte(password))
		seed, err := decryptSeed(encryptedSeed, passHash[:])
		returnIfErr(err)

		seedHash, err := getConfig(dataFile, []byte("HASH"))
		returnIfErr(err)

		if bytes.Compare(seedHash, dblSHA256(seed)) != 0 {
			returnIfErr(fmt.Errorf("incorrect password"))
		}

		w = wallet.New(seed, dataFile)

		coinAccount, err = w.CoinAccount(wallet.LTC, true, 0)
		returnIfErr(err)

		// TODO: Determine the discover dynamically
		d := discover.NewLitecoindLTCDiscover(
			"http://localhost:17001/", "btcuser1",
			"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
		)
		coinAccount.SetDiscover(d)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	ltcCmd.AddCommand(&cobra.Command{
		Use:   "balance",
		Short: "get balance of the wallet",
		Long:  `get balance of the wallet`,

		Run: func(cmd *cobra.Command, args []string) {
			bal, err := coinAccount.GetBalance()
			returnIfErr(err)
			fmt.Println("Balance: ", bal)
		},
	})

	ltcCmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "sync the wallet from the network",
		Long:  `sync the wallet from the network`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Sync data from network. It takes a period of time...")
			err := coinAccount.Discover()
			returnIfErr(err)
			bal, err := coinAccount.GetBalance()
			returnIfErr(err)
			fmt.Println("Balance: ", bal)
		},
	})

	ltcCmd.AddCommand(&cobra.Command{
		Use:   "newaddress",
		Short: "generate an used address of the wallet",
		Long:  `generate an used address of the wallet`,
		Run: func(cmd *cobra.Command, args []string) {
			addr, err := coinAccount.NewExternalAddr()
			returnIfErr(err)
			fmt.Println("Address: ", addr)
		},
	})

	var isHexData bool
	var data string
	sendCmd := &cobra.Command{
		Use:   "pay [address] [amount]",
		Short: "pay coin to an address",
		Long:  `pay coin to an address`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				cmd.Help()
				return
			}
			address := args[0]

			amount, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				returnIfErr(fmt.Errorf("invalid amount to send"))
			}

			var customData []byte
			if data != "" {
				if isHexData {
					customData, err = hex.DecodeString(data)
					returnIfErr(err)
				} else {
					customData = []byte(data)
				}
			}

			rawTx, err := coinAccount.Send(address, amount, customData)
			returnIfErr(err)
			fmt.Println("Raw Transaction: ", rawTx)
		},
	}
	sendCmd.Flags().BoolVarP(&isHexData, "hex-data", "H", false, "set the OP_RETURN data to be hex format")
	sendCmd.Flags().StringVarP(&data, "data", "D", "", "some custom data sent to OP_RETURN")

	ltcCmd.AddCommand(sendCmd)
}
