package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/bitgoin/tx"
	"github.com/bitmark-inc/bitmark-wallet/agent"
	"github.com/stretchr/testify/assert"
	"os"
)

const seedHex = "fded5e8970380eef15f742348d28511111366ae6a55188402b16c69922006fe6"

func TestWalletNew(t *testing.T) {
	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed, "wallet_test_new.dat")

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	t.Log("coin account:", ltcAccount)

	for i := uint32(0); i < 5; i++ {
		a, err := ltcAccount.Address(i, false)
		assert.NoError(t, err)
		t.Logf("external_%d: %s", i, a)
		a, err = ltcAccount.Address(i, true)
		t.Logf("internal_%d: %s", i, a)
	}
	os.Remove("wallet_test_new.dat")
}

func TestWalletAgent(t *testing.T) {
	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed, "wallet_test_discover.dat")

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	ltcAccount.SetAgent(agent.NewLitecoindAgent(
		"http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
	))

	err = ltcAccount.Discover()
	assert.NoError(t, err)
	utxos, err := ltcAccount.store.GetAllUTXO()
	assert.NotNil(t, utxos)
	assert.Len(t, utxos, 3)
	for _, txos := range utxos {
		for _, txo := range txos {
			t.Log(hex.EncodeToString(txo.TxHash), txo.Value, txo.TxIndex)
		}
	}

	os.Remove("wallet_test_discover.dat")
}

func TestWalletGetUTXO(t *testing.T) {
	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed, "wallet_test_getutxo.dat")

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	ltcAccount.SetAgent(agent.NewLitecoindAgent(
		"http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
	))

	err = ltcAccount.Discover()
	assert.NoError(t, err)
	utxos, err := ltcAccount.store.GetUTXO("mvxpcRGnjRpme59CAnLHTxFjwd8ivwWbQb")
	assert.NotNil(t, utxos)
	assert.Len(t, utxos, 2)
	os.Remove("wallet_test_getutxo.dat")
}

func TestWalletGetBalance(t *testing.T) {
	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed, "wallet_test_balance.dat")

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	ltcAccount.SetAgent(agent.NewLitecoindAgent(
		"http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
	))

	err = ltcAccount.Discover()
	assert.NoError(t, err)
	balance, err := ltcAccount.GetBalance()
	assert.NoError(t, err)
	t.Log(balance)
	i, err := ltcAccount.store.GetLastIndex()
	assert.NoError(t, err)
	t.Log(i)

	os.Remove("wallet_test_balances.dat")
}

func TestWalletGenCoins(t *testing.T) {
	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed, "wallet_test_gencoins.dat")

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	ltcAccount.SetAgent(agent.NewLitecoindAgent(
		"http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
	))

	err = ltcAccount.Discover()
	assert.NoError(t, err)
	coins1, amount1, err := ltcAccount.GenCoins(150000000)
	assert.NoError(t, err)
	t.Log("generate amount:", amount1)
	for _, txo := range coins1 {
		t.Log(hex.EncodeToString(txo.TxHash), txo.TxIndex, txo.Value)
	}
	assert.True(t, amount1 > 150000000)

	coins2, amount2, err := ltcAccount.GenCoins(275000000)
	assert.EqualError(t, err, "not enough of coins in the wallet")
	t.Log("generate amount:", amount2)
	assert.Nil(t, coins2)
	assert.True(t, amount2 < 275000000)
	os.Remove("wallet_test_gencoins.dat")
}

func TestWalletSend(t *testing.T) {
	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed, "wallet_test_send.dat")

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	ltcAccount.SetAgent(agent.NewLitecoindAgent(
		"http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw",
	))

	err = ltcAccount.Discover()
	assert.NoError(t, err)

	rawTx, err := ltcAccount.Send([]*tx.Send{{"mkeFURLRyDugRRP1kwKRcNBZwkVCPPmYkt", 155600000}}, nil, 0)
	t.Log(err)
	t.Log(rawTx)
}
