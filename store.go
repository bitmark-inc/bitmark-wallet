package wallet

import (
	"fmt"
	"time"

	"github.com/bitgoin/tx"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/boltdb/bolt"
)

var (
	ErrAccountNotExisted = fmt.Errorf("account is not existed")
)

func packUTXOs(utxos tx.UTXOs) []byte {
	b := make([]byte, 0)
	for _, utxo := range utxos {
		if utxo == nil {
			continue
		}
		hashLen := len(utxo.TxHash)
		b = append(b, util.ToVarint64(uint64(hashLen))...)
		b = append(b, utxo.TxHash...)
		b = append(b, util.ToVarint64(uint64(utxo.TxIndex))...)
		b = append(b, util.ToVarint64(utxo.Value)...)
	}
	return b
}

func unpackUTXOs(b []byte) tx.UTXOs {
	utxos := make([]*tx.UTXO, 0)
	offset := 0
	for offset < len(b) {
		txLen, txStart := util.FromVarint64(b[offset:])
		txEnd := txStart + int(txLen)
		txHash := b[offset+txStart : offset+txEnd]
		offset += txEnd
		txIndex, n := util.FromVarint64(b[offset:])
		offset += n
		val, n := util.FromVarint64(b[offset:])
		utxos = append(utxos, &tx.UTXO{
			TxHash:  txHash,
			TxIndex: uint32(txIndex),
			Value:   val,
		})
		offset += n
	}

	return utxos
}

type AccountStore interface {
	GetLastIndex() (uint64, error)
	SetLastIndex(uint64) error
	GetAllUTXO() (map[string]tx.UTXOs, error)
	GetUTXO(address string) (tx.UTXOs, error)
	SetUTXO(address string, utxo tx.UTXOs) error
	Close()
}

type BoltAccountStore struct {
	account string
	db      *bolt.DB
}

func (b BoltAccountStore) Close() {
	b.db.Close()
}

func (b BoltAccountStore) GetLastIndex() (uint64, error) {
	var buf []byte
	if err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.account))
		buf = bucket.Get([]byte("lastIndex"))
		return nil
	}); err != nil {
		return 0, err
	}
	index, _ := util.FromVarint64(buf)
	return index, nil
}

func (b BoltAccountStore) SetLastIndex(index uint64) error {
	buf := util.ToVarint64(index)
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.account))
		return bucket.Put([]byte("lastIndex"), buf)
	})
}

func (b BoltAccountStore) GetAllUTXO() (map[string]tx.UTXOs, error) {
	utxos := make(map[string]tx.UTXOs)
	if err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.account))
		if bucket == nil {
			return ErrAccountNotExisted
		}
		err := bucket.ForEach(func(address, tx []byte) error {
			txs := unpackUTXOs(tx)
			utxos[string(address)] = txs
			return nil
		})
		return err
	}); err != nil {
		return nil, err
	}
	return utxos, nil
}

func (b BoltAccountStore) GetUTXO(address string) (tx.UTXOs, error) {
	var utxos tx.UTXOs
	if err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.account))
		if bucket == nil {
			return ErrAccountNotExisted
		}
		utxos = unpackUTXOs(bucket.Get([]byte(address)))
		return nil
	}); err != nil {
		return nil, err
	}
	return utxos, nil
}

func (b BoltAccountStore) SetUTXO(address string, utxos tx.UTXOs) error {
	b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.account))
		if bucket == nil {
			return ErrAccountNotExisted
		}
		return bucket.Put([]byte(address), packUTXOs(utxos))
	})
	return nil
}

func NewBoltAccountStore(filename, account string) (*BoltAccountStore, error) {
	db, err := bolt.Open(filename, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(true)

	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.CreateBucketIfNotExists([]byte(account))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &BoltAccountStore{
		account: account,
		db:      db,
	}, nil
}
