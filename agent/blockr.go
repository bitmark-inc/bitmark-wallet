package agent

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"encoding/hex"
	"encoding/json"
	"github.com/bitgoin/tx"
	"net/url"
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

type BlockrAgent struct {
	apiHost string
	client  *http.Client
}

func (b BlockrAgent) GetAddrUnspent(addr string) ([]*tx.UTXO, error) {
	u := url.URL{
		Scheme: "https",
		Host:   b.apiHost,
		Path:   fmt.Sprintf("/api/v1/address/unspent/%s", addr),
	}
	r1, err := b.client.Get(u.String())
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
			hash, err := hex.DecodeString(u.Hash)
			if err != nil {
				return nil, err
			}

			utxo := tx.UTXO{
				TxHash:  reverseByte(hash),
				TxIndex: u.Index,
				Value:   uint64(u.Value * tx.Unit),
			}
			utxos = append(utxos, &utxo)
		}
		return utxos, nil
	}

	u.Path = fmt.Sprintf("/api/v1/address/info/%s", addr)
	r2, err := b.client.Get(u.String())
	if err != nil {
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

func NewBlockrAgent(apiHost string) *BlockrAgent {
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

	return &BlockrAgent{
		apiHost: apiHost,
		client:  c,
	}
}
