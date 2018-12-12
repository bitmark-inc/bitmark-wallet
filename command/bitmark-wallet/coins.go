package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bitmark-inc/bitmark-wallet"
	"github.com/bitmark-inc/bitmark-wallet/agent"
	"github.com/bitmark-inc/bitmark-wallet/tx"
)

var w *wallet.Wallet
var coinAccount *wallet.CoinAccount

var test bool

type AgentData struct {
	Type string
	Node string
	User string
	Pass string
}

func (a *AgentData) ParseFlag(flag *pflag.FlagSet) {
	// TODO: improve by struct tags
	if f := flag.Lookup("agent-type"); f != nil && f.Changed {
		a.Type = f.Value.String()
	}
	if f := flag.Lookup("agent-node"); f != nil && f.Changed {
		a.Node = f.Value.String()
	}
	if f := flag.Lookup("agent-user"); f != nil && f.Changed {
		a.User = f.Value.String()
	}
	if f := flag.Lookup("agent-pass"); f != nil && f.Changed {
		a.Pass = f.Value.String()
	}
}

func NewCoinCmd(coinType, short, long string, ct wallet.CoinType) *cobra.Command {
	var agentData AgentData
	cobra.OnInitialize(func() {
	agent_switch:
		switch v := viper.Get("agent").(type) {
		case []map[string]interface{}:
			if len(v) == 0 {
				break agent_switch
			}

			vv, ok := v[0][coinType]
			if !ok {
				break agent_switch
			}

			s := reflect.ValueOf(vv)
			if s.Kind() != reflect.Slice {
				break agent_switch
			}

			agentMap, ok := s.Index(0).Interface().(map[string]interface{})
			if !ok {
				break agent_switch
			}

			t, _ := agentMap["type"].(string)
			n, _ := agentMap["node"].(string)
			u, _ := agentMap["user"].(string)
			p, _ := agentMap["pass"].(string)

			agentData = AgentData{
				Type: t,
				Node: n,
				User: u,
				Pass: p,
			}
		case map[string]interface{}:
			err := viper.UnmarshalKey("agent", &agentData)
			if err != nil {
				fmt.Printf("Viper parser error: %s", err)
			}
		default:
			fmt.Println("Unexpected type agent value:", reflect.TypeOf(v))
			os.Exit(1)
		}
	})

	var cmd = &cobra.Command{
		Use:   coinType,
		Short: short,
		Long:  long,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// PreRun only for subcommand
			agentData.ParseFlag(cmd.Parent().PersistentFlags())

			if len(args) > 0 && args[0] == "help" {
				return
			}
			if cmd.Use == coinType {
				return
			}

			datadir := viper.GetString("datadir")
			walletdb := viper.GetString("walletdb")

			dataFile := path.Join(datadir, walletdb)
			if dataFile == "" {
				returnIfErr(fmt.Errorf("invalid wallet path"))
			}

			encryptedSeed, err := getWalletConfig(dataFile, []byte("SEED"))
			returnIfErr(err)

			password, err := readPassword("Input wallet password: ", 0)
			returnIfErr(err)

			passHash := dblSHA256([]byte(password))
			seed, err := decryptSeed(encryptedSeed, passHash[:])
			returnIfErr(err)

			seedHash, err := getWalletConfig(dataFile, []byte("HASH"))
			returnIfErr(err)

			if bytes.Compare(seedHash, dblSHA256(seed)) != 0 {
				returnIfErr(fmt.Errorf("incorrect password"))
			}

			w = wallet.New(seed, dataFile)

			coinAccount, err = w.CoinAccount(ct, wallet.Test(test), 0)
			returnIfErr(err)

			var a agent.CoinAgent
			switch agentData.Type {
			case "blockr":
				a = agent.NewBlockrAgent(agentData.Node)
			case "daemon":
				fallthrough
			default:
				url := fmt.Sprintf("http://%s/", agentData.Node)
				a = agent.NewDaemonAgent(url, agentData.User, agentData.Pass)
			}
			coinAccount.SetAgent(a)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	cmd.PersistentFlags().StringP("agent-type", "A", "daemon", "agent type of a wallet")
	cmd.PersistentFlags().StringP("agent-node", "N", "", "node of an agent")
	cmd.PersistentFlags().StringP("agent-user", "U", "", "user of an agent")
	cmd.PersistentFlags().StringP("agent-pass", "P", "", "password of an agent")

	cmd.PersistentFlags().BoolVarP(&test, "testnet", "t", false, "use the wallet in testnet")
	cmd.AddCommand(&cobra.Command{
		Use:   "balance",
		Short: "get balance of the wallet",
		Long:  `get balance of the wallet`,

		Run: func(cmd *cobra.Command, args []string) {
			bal, err := coinAccount.GetBalance()
			returnIfErr(err)
			fmt.Println("Balance: ", bal)
		},
	})

	cmd.AddCommand(&cobra.Command{
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

	cmd.AddCommand(&cobra.Command{
		Use:   "newaddress",
		Short: "generate an used address of the wallet",
		Long:  `generate an used address of the wallet`,
		Run: func(cmd *cobra.Command, args []string) {
			addr, err := coinAccount.NewExternalAddr()
			returnIfErr(err)
			fmt.Println("Address: ", addr)
		},
	})

	var fee uint64
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

			txId, rawTx, err := coinAccount.Send([]*tx.Send{{address, amount}}, customData, fee)
			returnIfErr(err)
			fmt.Printf(`{"txId": "%s", "rawTx": "%s"}`, txId, rawTx)
		},
	}
	sendCmd.Flags().StringVarP(&hexData, "hex-data", "H", "", "set hex bytes in the OP_RETURN")
	sendCmd.Flags().Uint64VarP(&fee, "fee", "f", 0, "set fee for per kB transaction.")
	cmd.AddCommand(sendCmd)

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
				if 2 != len(sendStrings) {
					returnIfErr(fmt.Errorf("argument must be 'address,satoshis'"))
				}
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

			txId, rawTx, err := coinAccount.Send(sends, customData, fee)
			returnIfErr(err)
			fmt.Printf(`{"txId": "%s", "rawTx": "%s"}`, txId, rawTx)
		},
	}
	sendManyCmd.Flags().StringVarP(&hexData, "hex-data", "H", "", "set hex bytes in the OP_RETURN")
	sendManyCmd.Flags().Uint64VarP(&fee, "fee", "f", 0, "set fee for per kB transaction.")
	cmd.AddCommand(sendManyCmd)
	return cmd
}
