package discover

import (
	"fmt"
	"github.com/bitgoin/tx"
)

var (
	ErrNoTxForAddr  = fmt.Errorf("no transaction for the address")
	ErrNoUnspentTx  = fmt.Errorf("no spentable transaction for the address")
	ErrQueryFailure = fmt.Errorf("fail to query from server")
)

type UTXODiscover interface {
	GetAddrUnspent(string) ([]*tx.UTXO, error)
}
