package agent

import (
	"fmt"
	"github.com/bitmark-inc/bitmark-wallet/tx"
)

var (
	ErrNoTxForAddr = fmt.Errorf("no transaction for the address")
)

type ErrQueryFailure struct {
	message string
}

func (e ErrQueryFailure) Error() string {
	return fmt.Sprintf("fail to query from server: %s", e.message)
}

type CoinAgent interface {
	ListAllUnspent() (map[string]tx.UTXOs, error)
	WatchAddress(addr string) error
	Send(string) (string, error)
}

func reverseByte(b []byte) []byte {
	l := len(b)
	newB := make([]byte, l)
	for i, x := range b {
		newB[l-i-1] = x
	}
	return newB
}
