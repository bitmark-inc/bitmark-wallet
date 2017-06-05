package wallet

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/bitgoin/address"
	"github.com/bitgoin/tx"
	"github.com/bitmark-inc/bitmark-wallet2/discover"
)

// Follow the rule of account discovery in BIP44
// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki#account-discovery
const AddressGap = 20

var (
	ErrNotEnoughCoin = fmt.Errorf("not enough of coins in the wallet")
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

// String returns the identifier of an account.
func (c CoinAccount) String() string {
	return c.identifier
}

type Wallet struct {
	seed []byte
}

func New(seed []byte) *Wallet {
	return &Wallet{
		seed: seed,
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

	store, err := NewBoltAccountStore("wallet.dat", pubkey.Address())
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
	externalKey, err := c.Key.Child(0)
	if err != nil {
		return err
	}

	var i uint32
	for i < AddressGap {
		k, err := externalKey.Child(i)
		if err != nil {
			return err
		}
		p, err := k.PubKey()
		if err != nil {
			return err
		}

		addr := p.Address()
		utxos, err := c.D.GetAddrUnspent(addr)
		if err != nil {
			if err == discover.ErrNoTxForAddr {
				i += 1
			} else {
				return err
			}
		} else {
			i = 0
		}
		err = c.store.SetUTXO(addr, utxos)
		if err != nil {
			return err
		}
	}
	return nil
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

	return coins, total, ErrNotEnoughCoin
}

func (c CoinAccount) Send(address string, amount uint64) error {

	changeAddr, err := c.NewChangeAddr()
	if err != nil {
		return err
	}

	// Generate the change address in advance.
	send := []*tx.Send{
		{
			Addr:   address,
			Amount: amount * tx.Unit,
		},
		{
			Addr:   changeAddr,
			Amount: 0,
		},
	}

	// Get UTXO recursively until the amount is greater than
	// sending amount
	coins, _, err := c.GenCoins(amount)
	if err != nil {
		return err
	}

	// TODO: calculate the transaction fee.
	ntx, used, err := tx.NewP2PKunsign(30000, coins, 0, send...)
	if err != nil {
	}

	err = tx.FillP2PKsign(ntx, used)
	log.Println(hex.EncodeToString(ntx.TxIn[0].Script))
	b, err := ntx.Pack()
	log.Println("raw tx:", hex.EncodeToString(b))
	return err
}
