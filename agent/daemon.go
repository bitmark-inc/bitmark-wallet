package agent

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bitmark-inc/bitmark-wallet/tx"
	log "github.com/sirupsen/logrus"
)

var (
	ErrImportAddress = fmt.Errorf("fail to import address")
)

var lastWatchedAddressesRefresh time.Time
var watchedAddressList []ReceivedAddress

type RPCParam struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type RPCUTXO struct {
	TxId    string  `json:"txid"`
	Address string  `json:"address"`
	Index   uint32  `json:"vout"`
	Value   float64 `json:"amount"`
}

type ReceivedAddress struct {
	Address string   `json:"address"`
	Amount  float64  `json:"amount"`
	TxIds   []string `json:"txids"`
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

func (da DaemonAgent) getAllWatchedAddress() error {
	if watchedAddressList != nil && time.Since(lastWatchedAddressesRefresh) < 15*time.Second {
		return nil
	}

	p := RPCParam{
		Method: "listreceivedbyaddress",
		Params: []interface{}{0, true, true},
	}

	v, err := da.jsonRPC(p)
	if err != nil {
		return err
	}

	err = json.Unmarshal(v.Result, &watchedAddressList)
	if err != nil {
		return err
	}

	lastWatchedAddressesRefresh = time.Now()
	log.WithField("timestamp", lastWatchedAddressesRefresh).Debug("refresh watched addresses")
	return err
}

func (da DaemonAgent) importAddress(addr string) error {
	p := RPCParam{
		Method: "importaddress",
		Params: []interface{}{addr, "bitmark-wallet watched", false},
	}
	_, err := da.jsonRPC(p)
	return err
}

func (da DaemonAgent) isAddressUsed(address string) (bool, error) {
	if len(watchedAddressList) == 0 {
		return false, nil
	}

	for _, r := range watchedAddressList {
		if r.Address == address && len(r.TxIds) > 0 {
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

func (da DaemonAgent) WatchAddress(addr string) error {
	err := da.getAllWatchedAddress()
	if err != nil {
		return fmt.Errorf("fail to update watched address: %s", err.Error())
	}

	addresses := map[string]bool{}
	for _, addr := range watchedAddressList {
		addresses[addr.Address] = true
	}

	if _, ok := addresses[addr]; !ok {
		err := da.importAddress(addr)
		if err != nil {
			return fmt.Errorf("fail to import address: %s", err.Error())
		}
		err = da.getAllWatchedAddress()
		if err != nil {
			return fmt.Errorf("fail to update watched address after import: %s", err.Error())
		}
	}

	// no unspent tx, check if it is an empty address
	if used, err := da.isAddressUsed(addr); err != nil {
		return err
	} else if !used {
		return ErrNoTxForAddr
	}
	return nil
}

func (da DaemonAgent) ListAllUnspent() (map[string]tx.UTXOs, error) {
	p := RPCParam{
		Method: "listunspent",
		Params: []interface{}{0, 999999},
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

	utxos := make(map[string]tx.UTXOs)

	for _, u := range rutxo {
		if u.Value == 0 {
			continue
		}

		hash, err := hex.DecodeString(u.TxId)
		if err != nil {
			return nil, err
		}

		if _, ok := utxos[u.Address]; !ok {
			utxos[u.Address] = make([]*tx.UTXO, 0)
		}

		utxos[u.Address] = append(utxos[u.Address], &tx.UTXO{
			TxHash:  reverseByte(hash),
			TxIndex: u.Index,
			Value:   uint64(u.Value * tx.Unit),
		})

	}

	return utxos, nil
}

func NewDaemonAgent(apiUrl, username, password string) *DaemonAgent {
	var t = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	var c = &http.Client{
		Timeout:   time.Second * 60,
		Transport: t,
	}

	return &DaemonAgent{
		apiUrl:   apiUrl,
		username: username,
		password: password,
		client:   c,
	}
}
