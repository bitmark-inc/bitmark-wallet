package wallet

import (
	"encoding/hex"
	"fmt"

	"github.com/bitgoin/address"
	"github.com/bitgoin/tx"
	"github.com/bitmark-inc/bitmark-wallet/discover"
)

// Follow the rule of account discovery in BIP44
// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki#account-discovery
const (
	TxFeePerKb = 100000
	AddressGap = 20
)

var (
	ErrNotEnoughCoin   = fmt.Errorf("not enough of coins in the wallet")
	ErrNilAccountStore = fmt.Errorf("no account store is set")
)

// CoinAccount is the root struct for manipulate coins.
type CoinAccount struct {
	CoinType   CoinType
	Test       Test
	Key        *address.ExtendedKey
	params     *address.Params
	D          discover.UTXODiscover
	store      AccountStore
	index      uint32
	identifier string
}

func calcTxFee(coins tx.UTXOs, opReturn *tx.TxOut, sends ...*tx.Send) (uint64, error) {
	// Set the maximum fee for calculation
	ntx, used, err := tx.NewP2PKunsign(0.1*tx.Unit, coins, 0, sends...)
	if err != nil {
		return 0, err
	}

	if opReturn != nil {
		ntx.TxOut = append(ntx.TxOut, opReturn)
	}

	if err = tx.FillP2PKsign(ntx, used); err != nil {
		return 0, err
	}

	rawTx, err := ntx.Pack()
	if err != nil {
		return 0, err
	}
	return uint64(len(rawTx)) * TxFeePerKb / 1000, nil
}

func (c CoinAccount) prepareTx(coins tx.UTXOs, customData []byte, sends ...*tx.Send) (*tx.Tx, tx.UTXOs, error) {
	var opReturn *tx.TxOut
	if customData != nil {
		opReturn = tx.CustomTx(customData)
	}
	fee, err := calcTxFee(coins, opReturn, sends...)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("Fee: ", fee)
	ntx, used, err := tx.NewP2PKunsign(fee, coins, 0, sends...)
	if err != nil {
		return nil, nil, err
	}

	if opReturn != nil {
		ntx.TxOut = append(ntx.TxOut, opReturn)
	}

	return ntx, used, nil
}

// String returns the identifier of an account.
func (c CoinAccount) String() string {
	return c.identifier
}

type Wallet struct {
	seed     []byte
	dataFile string
}

func New(seed []byte, dataFile string) *Wallet {
	return &Wallet{
		seed:     seed,
		dataFile: dataFile,
	}
}

// CoinAccount returns an extended account base on BIP44 with
// the coin type and the account index being specified.
func (w Wallet) CoinAccount(ct CoinType, test Test, account uint32) (*CoinAccount, error) {
	coinParams := CoinParams[ct][test]
	masterKey, err := address.NewMaster(w.seed, coinParams)
	if err != nil {
		return nil, err
	}
	// m / 44'
	bip44Key, err := masterKey.Child(44)
	if err != nil {
		return nil, err
	}

	// m / 44' / ct'
	cointKey, err := bip44Key.Child(CoinMap[ct])
	if err != nil {
		return nil, err
	}

	// m / 44' / coin' / account'
	accountKey, err := cointKey.Child(account)
	if err != nil {
		return nil, err
	}

	pubkey, err := accountKey.PubKey()
	if err != nil {
		return nil, err
	}

	store, err := NewBoltAccountStore(w.dataFile, pubkey.Address())
	if err != nil {
		return nil, err
	}

	return &CoinAccount{
		CoinType:   ct,
		Test:       test,
		Key:        accountKey,
		store:      store,
		params:     coinParams,
		identifier: pubkey.Address(),
	}, nil
}

func (c *CoinAccount) SetDiscover(d discover.UTXODiscover) {
	c.D = d
}

func (c CoinAccount) addressKey(i uint32, change bool) (*address.PrivateKey, error) {
	var changeBit uint32
	if change {
		changeBit = 1
	}
	externalKey, err := c.Key.Child(changeBit)
	if err != nil {
		return nil, err
	}

	addreseAccount, err := externalKey.Child(i)
	if err != nil {
		return nil, err
	}

	return addreseAccount.PrivKey()
}

func (c CoinAccount) NewChangeAddr() (string, error) {
	lastIndex, err := c.store.GetLastIndex()
	if err != nil {
		return "", err
	}
	return c.Address(uint32(lastIndex), true)
}

func (c CoinAccount) NewExternalAddr() (string, error) {
	lastIndex, err := c.store.GetLastIndex()
	if err != nil {
		return "", err
	}
	return c.Address(uint32(lastIndex), false)
}

// Address returns a coin address
func (c CoinAccount) Address(i uint32, change bool) (string, error) {
	p, err := c.addressKey(i, change)
	if err != nil {
		return "", err
	}

	return p.PublicKey.Address(), nil
}

func (c CoinAccount) Discover() error {
	// m / 44' / coin' / account' / external
	var lastIndex uint64
	for i := uint32(0); i < 2; i++ {
		changeKey, err := c.Key.Child(i)
		if err != nil {
			return err
		}
		var gap, j uint32
		for gap < AddressGap {
			k, err := changeKey.Child(j)
			if err != nil {
				return err
			}

			p, err := k.PubKey()
			if err != nil {
				return err
			}

			addr := p.Address()
			utxos, err := c.D.GetAddrUnspent(addr)

			switch err {
			case discover.ErrNoTxForAddr:
				gap += 1
			case nil, discover.ErrNoUnspentTx:
				gap = 0
				if i == 0 { // that means external address
					lastIndex += 1
				}
			default:
				return err
			}

			err = c.store.SetUTXO(addr, utxos)
			if err != nil {
				return err
			}
			j += 1
		}
	}
	return c.store.SetLastIndex(lastIndex)
}

func (c CoinAccount) GetBalance() (uint64, error) {
	utxos, err := c.store.GetAllUTXO()
	if err != nil {
		return 0, err
	}
	var balance uint64
	for _, txos := range utxos {
		for _, txo := range txos {
			balance += txo.Value
		}
	}
	return balance, nil
}

// GenCoins will generate coins for sending in address order
func (c CoinAccount) GenCoins(amount uint64) (tx.UTXOs, uint64, error) {
	coins := make([]*tx.UTXO, 0)
	var total uint64
	utxos, err := c.store.GetAllUTXO()
	if err != nil {
		return nil, 0, err
	}

	l, err := c.store.GetLastIndex()
	if err != nil {
		return nil, 0, err
	}
	// Use changes first
	for j := 1; j >= 0; j-- {
		for i := uint32(0); i <= uint32(l); i++ {
			p, err := c.addressKey(i, j == 1) // 0: external, 1: internal(changes)
			if err != nil {
				return nil, 0, err
			}
			address := p.PublicKey.Address()
			if txs, ok := utxos[address]; ok {
				script, err := tx.DefaultP2PKScript(address)
				if err != nil {
					return nil, 0, err
				}
				for i := 0; i < len(txs); i++ {
					u := txs[i]
					u.Key = p
					u.Script = script
					coins = append(coins, u)
					total += u.Value
				}
			}
			if total >= amount {
				return coins, total, nil
			}
		}
	}

	return nil, total, ErrNotEnoughCoin
}

func (c CoinAccount) Send(sends []*tx.Send, customData []byte) (string, error) {
	// Generate the change address in advance.
	changeAddr, err := c.NewChangeAddr()
	if err != nil {
		return "", err
	}

	sends = append(sends, &tx.Send{
		Addr:   changeAddr,
		Amount: 0,
	})

	var amounts uint64

	for _, s := range sends {
		amounts += s.Amount
	}
	// Get UTXO recursively until the amount is greater than
	// sending amount
	coins, _, err := c.GenCoins(amounts)
	if err != nil {
		return "", err
	}

	ntx, used, err := c.prepareTx(coins, customData, sends...)
	if err != nil {
		return "", err
	}
	err = tx.FillP2PKsign(ntx, used)
	if err != nil {
		return "", err
	}
	b, err := ntx.Pack()
	if err != nil {
		return "", err
	}
	return c.D.Send(hex.EncodeToString(b))
}
