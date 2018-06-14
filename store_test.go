package wallet

import (
	"os"
	"testing"

	"github.com/bitgoin/tx"
	"github.com/stretchr/testify/assert"
)

func TestUnpackEmptyBuffer(t *testing.T) {
	utxos := unpackUTXOs([]byte{})
	assert.Len(t, utxos, 0)
}

func TestPackEmptyUTXO(t *testing.T) {
	utxos := make([]*tx.UTXO, 0)
	b := packUTXOs(utxos)
	assert.Len(t, b, 0)
}

func TestBoltAccountStoreGetAndSet(t *testing.T) {
	test_utxos := []*tx.UTXO{
		{
			TxHash:  []byte("fakehash"),
			TxIndex: 0,
			Value:   100000,
		},
	}
	s, err := NewBoltAccountStore("wallet_test.dat", "test_account")
	assert.NoError(t, err)
	err = s.SetUTXO("fakeaddr", test_utxos)
	assert.NoError(t, err)
	utxos, err := s.GetUTXO("fakeaddr")
	assert.NoError(t, err)
	assert.Len(t, utxos, 1)
	u := utxos[0]
	assert.Equal(t, u.TxHash, []byte("fakehash"))
	assert.Equal(t, u.TxIndex, uint32(0))
	assert.Equal(t, u.Value, uint64(100000))
	os.Remove("wallet_test.dat")
}

func TestBoltAccountStoreGetAll(t *testing.T) {
	test_utxos := []*tx.UTXO{
		{
			TxHash:  []byte("fakehash"),
			TxIndex: 0,
			Value:   100000,
		},
		{
			TxHash:  []byte("fakehash1"),
			TxIndex: 1,
			Value:   200000,
		},
	}
	s, err := NewBoltAccountStore("wallet_test2.dat", "test_account")
	assert.NoError(t, err)
	err = s.SetUTXO("fakeaddr", test_utxos)
	assert.NoError(t, err)
	utxos, err := s.GetAllUTXO()
	assert.NoError(t, err)

	txos, ok := utxos["fakeaddr"]
	assert.True(t, ok)
	assert.Len(t, txos, 2)

	u0 := txos[0]
	assert.Equal(t, u0.TxHash, []byte("fakehash"))
	assert.Equal(t, u0.TxIndex, uint32(0))
	assert.Equal(t, u0.Value, uint64(100000))

	u1 := txos[1]
	assert.Equal(t, u1.TxHash, []byte("fakehash1"))
	assert.Equal(t, u1.TxIndex, uint32(1))
	assert.Equal(t, u1.Value, uint64(200000))
	os.Remove("wallet_test2.dat")
}
