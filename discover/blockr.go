package discover

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"encoding/hex"
	"encoding/json"
	"github.com/bitgoin/tx"
)

type BlockrUTXO struct {
	Hash  string  `json:"tx"`
	Index uint32  `json:"n"`
	Value float64 `json:"amount,string"`
}

type BlockrTxData struct {
	Tx    string `json:"tx"`
	Block int64  `json:"block_nb,string"`
}

type BlockrUnspentData struct {
	Address string       `json:"address"`
	Unspent []BlockrUTXO `json:"unspent"`
}

type BlockrAddrData struct {
	Address string        `json:"address"`
	Balance float64       `json:"balance"`
	FirstTx *BlockrTxData `json:"first_tx"`
}

type BlockrAddrInfoResp struct {
	Status string         `json:"status"`
	Data   BlockrAddrData `json:"data"`
}

type BlockrUnspentResp struct {
	Status string            `json:"status"`
	Data   BlockrUnspentData `json:"data"`
}

type BlockrBtcDiscover struct {
	client *http.Client
}

func (b BlockrBtcDiscover) GetAddrUnspent(addr string) ([]*tx.UTXO, error) {
	unspentQuery := fmt.Sprintf("http://btc.blockr.io/api/v1/address/unspent/%s", addr)
	r1, err := b.client.Get(unspentQuery)
	if err != nil {
		return nil, ErrQueryFailure
	}
	defer r1.Body.Close()

	var v1 BlockrUnspentResp
	d1 := json.NewDecoder(r1.Body)
	err = d1.Decode(&v1)
	if err != nil {
		return nil, err
	}

	if len(v1.Data.Unspent) != 0 {
		utxos := make([]*tx.UTXO, 0, len(v1.Data.Unspent))
		for _, u := range v1.Data.Unspent {
			hashByte, err := hex.DecodeString(u.Hash)
			if err != nil {
				return nil, err
			}
			utxo := tx.UTXO{
				TxHash:  hashByte,
				TxIndex: u.Index,
				Value:   uint64(u.Value * tx.Unit),
			}
			utxos = append(utxos, &utxo)
		}
		return utxos, nil
	}

	addrQuery := fmt.Sprintf("http://btc.blockr.io/api/v1/address/info/%s", addr)
	r2, err := b.client.Get(addrQuery)
	if err != nil {
		return nil, ErrQueryFailure
	}
	defer r2.Body.Close()

	var v2 BlockrAddrInfoResp
	d2 := json.NewDecoder(r2.Body)
	err = d2.Decode(&v2)
	if err != nil {
		return nil, err
	}

	if v2.Data.FirstTx == nil {
		return nil, ErrNoTxForAddr
	}

	return nil, ErrNoUnspentTx
}

func NewBlockrBtcDiscover() *BlockrBtcDiscover {
	var t = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	var c = &http.Client{
		Timeout:   time.Second * 10,
		Transport: t,
	}

	return &BlockrBtcDiscover{
		client: c,
	}
}
