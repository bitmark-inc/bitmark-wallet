package main

import (
	"encoding/hex"
	"flag"
	"time"

	wallet "github.com/bitmark-inc/bitmark-wallet"
	"github.com/bitmark-inc/bitmark-wallet/agent"
	log "github.com/sirupsen/logrus"
)

func main() {
	var seed, walletdb, apiToken string
	var rpcconnect, rpcuser, rpcpassword string

	flag.StringVar(&seed, "seed", "", "hd wallet seed")
	flag.StringVar(&walletdb, "walletdb", "wallet.dat", "hd wallet db")
	flag.StringVar(&apiToken, "api-token", "", "server api-token")
	flag.StringVar(&rpcconnect, "rpcconnect", "http://127.0.0.1:8332", "bitcoind RPC connect")
	flag.StringVar(&rpcuser, "rpcuser", "bitmark", "bitcoind RPC user")
	flag.StringVar(&rpcpassword, "rpcpassword", "", "bitcoind RPC password")
	flag.Parse()

	// log.SetLevel(log.DebugLevel)

	b, err := hex.DecodeString(seed)
	if err != nil {
		log.WithError(err).WithField("seed", seed).Panic("invalid seed")
	}
	w := wallet.New(b, walletdb)

	// force to be testnet
	coinAccount, err := w.CoinAccount(wallet.BTC, wallet.Test(true), 0)
	if err != nil {
		panic(err)
	}

	coinAccount.SetAgent(
		agent.NewDaemonAgent(rpcconnect, rpcuser, rpcpassword),
	)

	for {
		time.Sleep(2 * time.Second)
		if err := coinAccount.Discover(); err != nil {
			log.WithError(err).Error("discover")
		}

		coinAccount.GetBalance()

		addr, err := coinAccount.NewExternalAddr()
		if err != nil {
			log.WithError(err).Error("new external address")
		}
		log.WithField("address", addr).Info("new external address")
	}
}
