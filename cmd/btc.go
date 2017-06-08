package main

import (
	"bytes"
	"fmt"
	"path"
	"strconv"

	"encoding/hex"
	"github.com/bitgoin/tx"
	"github.com/bitmark-inc/bitmark-wallet"
	"github.com/bitmark-inc/bitmark-wallet/agent"
	"github.com/spf13/cobra"
	"strings"
)

var btcCmd = &cobra.Command{
	Use:   "btc",
	Short: "Bitcoin wallet",
	Long:  `Bitcoin wallet`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// PreRun only for subcommand
		if len(args) > 0 && args[0] == "help" {
			return
		}
		if cmd.Use == "btc" {
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

		coinAccount, err = w.CoinAccount(wallet.BTC, wallet.Test(test), 0)
		returnIfErr(err)

		// TODO: Determine the discover dynamically
		d := agent.NewLitecoindAgent(
			"http://localhost:17001/", "btcuser1",
			"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
		)
		coinAccount.SetAgent(d)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	btcCmd.PersistentFlags().BoolVarP(&test, "testnet", "t", false, "use the wallet in testnet")
	btcCmd.AddCommand(&cobra.Command{
		Use:   "balance",
		Short: "get balance of the wallet",
		Long:  `get balance of the wallet`,

		Run: func(cmd *cobra.Command, args []string) {
			bal, err := coinAccount.GetBalance()
			returnIfErr(err)
			fmt.Println("Balance: ", bal)
		},
	})

	btcCmd.AddCommand(&cobra.Command{
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

	btcCmd.AddCommand(&cobra.Command{
		Use:   "newaddress",
		Short: "generate an used address of the wallet",
		Long:  `generate an used address of the wallet`,
		Run: func(cmd *cobra.Command, args []string) {
			addr, err := coinAccount.NewExternalAddr()
			returnIfErr(err)
			fmt.Println("Address: ", addr)
		},
	})

	var hexData string
	sendCmd := &cobra.Command{
		Use:   "send [address] [amount]",
		Short: "send coins to an address",
		Long:  `send coins to an address`,
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
			if hexData != "" {
				customData, err = hex.DecodeString(hexData)
				returnIfErr(err)
			}

			err = coinAccount.Discover()
			returnIfErr(err)

			rawTx, err := coinAccount.Send([]*tx.Send{{address, amount}}, customData)
			returnIfErr(err)
			fmt.Println("Raw Transaction: ", rawTx)
		},
	}
	sendCmd.Flags().StringVarP(&hexData, "hex-data", "H", "", "set hex bytes in the OP_RETURN")
	btcCmd.AddCommand(sendCmd)

	sendManyCmd := &cobra.Command{
		Use:   "sendmany [address,amount] [address,amount] ...",
		Short: "send coins to an address",
		Long:  `send coins to an address`,
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			if len(args) < 1 {
				cmd.Help()
				return
			}

			sends := []*tx.Send{}
			for _, s := range args {
				sendStrings := strings.Split(s, ",")
				addr := sendStrings[0]
				amount, err := strconv.ParseUint(sendStrings[1], 10, 64)
				if err != nil {
					returnIfErr(fmt.Errorf("invalid amount to send"))
				}

				send := &tx.Send{Addr: addr, Amount: amount}
				sends = append(sends, send)
			}

			var customData []byte
			if hexData != "" {
				customData, err = hex.DecodeString(hexData)
				returnIfErr(err)
			}

			err = coinAccount.Discover()
			returnIfErr(err)

			rawTx, err := coinAccount.Send(sends, customData)
			returnIfErr(err)
			fmt.Println("Raw Transaction: ", rawTx)
		},
	}
	sendManyCmd.Flags().StringVarP(&hexData, "hex-data", "H", "", "set hex bytes in the OP_RETURN")
	btcCmd.AddCommand(sendManyCmd)
}
