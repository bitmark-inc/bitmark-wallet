package agent

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/bitgoin/tx"
)

var (
	ErrImportAddress = fmt.Errorf("fail to import address")
)

type RPCParam struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type RPCUTXO struct {
	TxId  string  `json:"txid"`
	Index uint32  `json:"vout"`
	Value float64 `json:"amount"`
}

type ReceivedAddress struct {
	Address string `json:"address"`
}

type RPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

type DaemonAgent struct {
	apiUrl   string
	username string
	password string
	client   *http.Client
}

func (da DaemonAgent) jsonRPC(p RPCParam) (*RPCResponse, error) {
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	err := e.Encode(p)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", da.apiUrl, &buf)
	req.SetBasicAuth(da.username, da.password)

	r, err := da.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var v *RPCResponse
	d := json.NewDecoder(r.Body)
	err = d.Decode(&v)
	if err != nil {
		return nil, err
	}

	if v.Error != nil {
		return nil, fmt.Errorf("JSONRPC Error: %s (code: %d)", v.Error.Message, v.Error.Code)
	}
	return v, nil
}

func (da DaemonAgent) importAddress(addr string) error {
	p := RPCParam{
		Method: "importaddress",
		Params: []interface{}{addr, "", false},
	}
	_, err := da.jsonRPC(p)
	return err
}

func (da DaemonAgent) listUnspent(addr string) (tx.UTXOs, error) {
	utxos := make([]*tx.UTXO, 0)
	p := RPCParam{
		Method: "listunspent",
		Params: []interface{}{0, 999999, []string{addr}},
	}
	v, err := da.jsonRPC(p)
	if err != nil {
		return nil, err
	}

	var rutxo []RPCUTXO
	err = json.Unmarshal(v.Result, &rutxo)
	if err != nil {
		return nil, err
	}

	for _, u := range rutxo {
		hash, err := hex.DecodeString(u.TxId)
		if err != nil {
			return nil, err
		}

		utxos = append(utxos, &tx.UTXO{
			TxHash:  reverseByte(hash),
			TxIndex: u.Index,
			Value:   uint64(u.Value * tx.Unit),
		})
	}

	return utxos, nil
}

func (da DaemonAgent) isAddressUsed(address string) (bool, error) {
	p := RPCParam{
		Method: "listreceivedbyaddress",
		Params: []interface{}{1, false, true},
	}

	v, err := da.jsonRPC(p)
	if err != nil {
		return false, err
	}

	var received []ReceivedAddress
	err = json.Unmarshal(v.Result, &received)
	if err != nil {
		return false, err
	}

	for _, r := range received {
		if r.Address == address {
			return true, nil
		}
	}
	return false, nil
}

func (da DaemonAgent) Send(rawTx string) (string, error) {
	p := RPCParam{
		Method: "sendrawtransaction",
		Params: []interface{}{rawTx},
	}

	v, err := da.jsonRPC(p)
	if err != nil {
		return "", err
	}

	var txId string
	err = json.Unmarshal(v.Result, &txId)
	if err != nil {
		return "", err
	}

	return txId, nil
}

func (da DaemonAgent) GetAddrUnspent(addr string) ([]*tx.UTXO, error) {
	err := da.importAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("fail to import address: %s", err.Error())
	}

	utxos, err := da.listUnspent(addr)
	if len(utxos) > 0 {
		return utxos, nil
	}

	// no unspent tx, check if it is an empty address
	if used, err := da.isAddressUsed(addr); err != nil {
		return nil, err
	} else if !used {
		return nil, ErrNoTxForAddr
	}
	return nil, ErrNoUnspentTx
}

func NewDaemonAgent(apiUrl, username, password string) *DaemonAgent {
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

	return &DaemonAgent{
		apiUrl:   apiUrl,
		username: username,
		password: password,
		client:   c,
	}
}
