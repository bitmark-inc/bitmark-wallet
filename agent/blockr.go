package agent

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/bitmark-inc/bitmark-wallet/tx"
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

type BlockrResp struct {
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Code    int             `json:"code"`
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
		return nil, ErrQueryFailure{err.Error()}
	}
	defer r1.Body.Close()

	var v1 BlockrResp
	d1 := json.NewDecoder(r1.Body)
	err = d1.Decode(&v1)
	if err != nil {
		return nil, err
	}

	var unspentData BlockrUnspentData
	err = json.Unmarshal(v1.Data, &unspentData)
	if err != nil {
		return nil, err
	}

	if len(unspentData.Unspent) != 0 {
		utxos := make([]*tx.UTXO, 0, len(unspentData.Unspent))
		for _, u := range unspentData.Unspent {
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
		return nil, ErrQueryFailure{err.Error()}
	}
	defer r2.Body.Close()

	var v2 BlockrResp
	d2 := json.NewDecoder(r2.Body)
	err = d2.Decode(&v2)
	if err != nil {
		return nil, err
	}

	var addrData BlockrAddrData
	err = json.Unmarshal(v2.Data, &addrData)
	if err != nil {
		return nil, err
	}

	if addrData.FirstTx == nil {
		return nil, ErrNoTxForAddr
	}

	return nil, ErrNoUnspentTx
}

func (b BlockrAgent) Send(rawTx string) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   b.apiHost,
		Path:   fmt.Sprintf("/api/v1/tx/push"),
	}

	var buf bytes.Buffer
	v := map[string]string{"hex": rawTx}
	e := json.NewEncoder(&buf)
	err := e.Encode(v)
	if err != nil {
		return "", err
	}

	r, err := b.client.Post(u.String(), "application/json", &buf)
	if err != nil {
		return "", ErrQueryFailure{err.Error()}
	}
	defer r.Body.Close()

	var ret BlockrResp
	d := json.NewDecoder(r.Body)
	err = d.Decode(&ret)
	if err != nil {
		return "", nil
	}

	if r.StatusCode == 200 {
		return string(ret.Data), nil
	} else {
		return "", fmt.Errorf("%s (%s)", ret.Data, ret.Message)
	}
}

func NewBlockrAgent(apiHost string) *BlockrAgent {
	var t = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	var c = &http.Client{
		Timeout:   time.Second * 20,
		Transport: t,
	}

	return &BlockrAgent{
		apiHost: apiHost,
		client:  c,
	}
}
