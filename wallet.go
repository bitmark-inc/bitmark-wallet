package wallet

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/bitgoin/address"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	log "github.com/sirupsen/logrus"

	"github.com/bitmark-inc/bitmark-wallet/agent"
	"github.com/bitmark-inc/bitmark-wallet/tx"
)

// Follow the rule of account discovery in BIP44
// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki#account-discovery
const (
	AddressGap = 5
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
	agent      agent.CoinAgent
	store      AccountStore
	feePerKB   uint64
	index      uint32
	identifier string
}

func (c *CoinAccount) Close() {
	c.store.Close()
}

// signTx adds signatureScripts for each vins
func (c CoinAccount) signTx(utxos tx.UTXOs, redeemTx *wire.MsgTx) error {
	for i := range redeemTx.TxIn {
		utxo := utxos[i]

		signatureScript, err := txscript.SignatureScript(redeemTx, i, utxo.Script, txscript.SigHashAll, (*btcec.PrivateKey)(utxo.Key.PrivateKey), true)
		if err != nil {
			return err
		}

		redeemTx.TxIn[i].SignatureScript = signatureScript
	}

	return nil
}

type UnspentFunds struct {
	TxIn        []*wire.TxIn
	TotalAmount uint64
	UTXOs       tx.UTXOs
}

// prepareUnspentFunds returns the UnspentFunds which includes vins, total amounts and
// signing information of each vins
func (c CoinAccount) prepareUnspentFunds(amount uint64) (*UnspentFunds, error) {
	utxos, total, err := c.collectUTXOs(amount)
	if err != nil {
		return nil, err
	}

	if total < amount {
		return nil, fmt.Errorf("no enough money to spend")
	}

	txInputs := []*wire.TxIn{}
	for _, u := range utxos {
		utxoHash, err := chainhash.NewHash(u.TxHash)
		if err != nil {
			return nil, err
		}

		outPoint := wire.NewOutPoint(utxoHash, u.TxIndex)
		txIn := wire.NewTxIn(outPoint, nil, nil)

		txInputs = append(txInputs, txIn)
	}

	return &UnspentFunds{
		TxIn:        txInputs,
		TotalAmount: total,
		UTXOs:       utxos,
	}, nil
}

// prepareSpendTx creates a transaction by collecting enough vins,
// adding vouts for destination and signing the transaction
func (c CoinAccount) prepareSpendTx(customData []byte, sends []*tx.Send, changeAddr string, feePerKB uint64) (*wire.MsgTx, error) {
	redeemTx := wire.NewMsgTx(wire.TxVersion)

	var totalInputAmount, totalOutputAmount uint64

	var initialAmount uint64
	for _, s := range sends {
		initialAmount += s.Amount
	}

	unspentFunds, err := c.prepareUnspentFunds(initialAmount)
	if err != nil {
		return nil, err
	}
	redeemTx.TxIn = unspentFunds.TxIn
	totalInputAmount = unspentFunds.TotalAmount

	// prepare change pkScript
	decodedChangeAddr, err := btcutil.DecodeAddress(changeAddr, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, err
	}
	changePKScript, err := txscript.PayToAddrScript(decodedChangeAddr)
	if err != nil {
		return nil, err
	}

	totalVout := len(sends)

	for _, s := range sends {
		decodedAddr, err := btcutil.DecodeAddress(s.Addr, &chaincfg.TestNet3Params)
		if err != nil {
			return nil, err
		}
		destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
		if err != nil {
			return nil, err
		}

		totalOutputAmount += s.Amount
		redeemTxOut := wire.NewTxOut(int64(s.Amount), destinationAddrByte)
		redeemTx.AddTxOut(redeemTxOut)
	}

	// add custome data as the last vout
	if customData != nil {
		builder := txscript.NewScriptBuilder()
		script, err := builder.AddOp(txscript.OP_RETURN).AddData(customData).Script()
		if err != nil {
			return nil, err
		}
		totalVout += 1
		redeemTx.AddTxOut(wire.NewTxOut(0, script))
	}

	// add scriptSig into transaction for better fee estimation
	if err := c.signTx(unspentFunds.UTXOs, redeemTx); err != nil {
		return nil, err
	}

	txSize := 0
	// fee estimation loop
	for {
		log.WithField("txSize", txSize).Info("compare tx size")
		if redeemTx.SerializeSize() <= txSize {
			break
		}
		txSize = redeemTx.SerializeSize()

		feePerByte := int(feePerKB) / 1000

		newFee := int64(txSize * feePerByte)
		changeAmount := int64(totalInputAmount) - int64(totalOutputAmount) - newFee
		log.WithField("fee", newFee).WithField("change", changeAmount).Info("estimate change value")
		if changeAmount < 0 {
			// changeAmount is less than zero which indicates that the fee is not enough
			newInputAmount := totalInputAmount - uint64(changeAmount)
			log.WithField("newInputAmount", newInputAmount).Info("not enough of transaction fee. need for input")

			var err error
			unspentFunds, err = c.prepareUnspentFunds(newInputAmount)
			if err != nil {
				return nil, err
			}
			redeemTx.TxIn = unspentFunds.TxIn
			totalInputAmount = unspentFunds.TotalAmount

			// reset the evaluated txSize
			txSize = 0
		} else if changeAmount > 35*int64(feePerByte) {
			// add the change vout only when the change is greater than 35 * feePerByte
			// this value is determined by the extra transaction sizes of an addition vout

			if len(redeemTx.TxOut) == totalVout {
				// add change vout as the first item
				redeemTxOut := wire.NewTxOut(0, changePKScript)
				fmt.Println(redeemTxOut.SerializeSize())
				redeemTx.TxOut = append([]*wire.TxOut{redeemTxOut}, redeemTx.TxOut...)
			}
			redeemTx.TxOut[0].Value = changeAmount
		}

		if err := c.signTx(unspentFunds.UTXOs, redeemTx); err != nil {
			return nil, err
		}
	}

	return redeemTx, nil
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
		feePerKB:   CoinFee[ct],
		identifier: pubkey.Address(),
	}, nil
}

func (c *CoinAccount) SetAgent(a agent.CoinAgent) {
	c.agent = a
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
	return c.Address(uint32(lastIndex)+1, false)
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
	addresses := make([]string, 0)

	// m / 44' / coin' / account' / external
	var lastIndex uint64
	for i := uint32(0); i < 2; i++ { // i = 0 external, i = 1 internal (change)
		changeKey, err := c.Key.Child(i)
		if err != nil {
			return err
		}
		var gap, j uint32
		var _lastIndex uint64
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
			err = c.agent.WatchAddress(addr)
			switch err {
			case agent.ErrNoTxForAddr:
				gap += 1
			case nil:
				gap = 0
				// Update the _lastIndex if there are transactions found
				if i == 0 {
					log.WithField("address", addr).WithField("index", j).Debug("discover external transactions")
				}
				_lastIndex = uint64(j)
			default:
				return err
			}

			addresses = append(addresses, addr)
			j += 1
		}
		log.WithField("external", i == 0).WithField("lastIndex", _lastIndex).Debug("discovered last index")
		// make sure the last index is largest number between external and internal
		if _lastIndex > lastIndex {
			lastIndex = _lastIndex
		}
	}

	addrUTXOs, err := c.agent.ListAllUnspent()
	if err != nil {
		return err
	}

	for _, addr := range addresses {
		// addrUTXOs[addr] might
		// 1. contain utxos, and then the entry will be updated
		// 2. NOT exist, and then the entry will be deleted
		err = c.store.SetUTXO(addr, addrUTXOs[addr])
		if err != nil {
			return err
		}
	}
	log.WithField("lastIndex", lastIndex).Debug("set last index")
	return c.store.SetLastIndex(lastIndex)
}

func (c CoinAccount) GetBalance() (uint64, error) {
	utxos, err := c.store.GetAllUTXO()
	if err != nil {
		return 0, err
	}
	var balance uint64
	for addr, txos := range utxos {
		for _, txo := range txos {
			log.
				WithField("address", addr).
				WithField("tx", hex.EncodeToString(txo.TxHash)).
				WithField("amount", txo.Value).
				Info("unspent fund")
			balance += txo.Value
		}
	}
	return balance, nil
}

// collectUTXOs will collect UTXOs to fulfill a given amount
func (c CoinAccount) collectUTXOs(amount uint64) (tx.UTXOs, uint64, error) {
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
COLLECT_UTXOS:
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

					if total >= amount {
						break COLLECT_UTXOS
					}
				}
			}
		}
	}
	return coins, total, nil
}

func (c CoinAccount) Send(sends []*tx.Send, customData []byte, fee uint64) (string, string, error) {
	feePerKB := c.feePerKB
	if fee != 0 {
		feePerKB = fee
	}
	// Generate the change address in advance.
	changeAddr, err := c.NewChangeAddr()
	if err != nil {
		return "", "", err
	}

	redeemTx, err := c.prepareSpendTx(customData, sends, changeAddr, feePerKB)
	if err != nil {
		return "", "", err
	}

	var signedTx bytes.Buffer
	if err := redeemTx.Serialize(&signedTx); err != nil {
		return "", "", err
	}
	rawTx := hex.EncodeToString(signedTx.Bytes())
	txId, err := c.agent.Send(rawTx)
	if err != nil {
		log.WithError(err).WithField("rawTx", rawTx).Error("unable to broadcast transaction")
		return "", "", err
	}

	return txId, rawTx, err
}
