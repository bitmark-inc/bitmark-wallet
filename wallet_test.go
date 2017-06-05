package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

const seedHex = "fded5e8970380eef15f742348d28511111366ae6a55188402b16c69922006fe6"

func TestWalletNew(t *testing.T) {

	seed, err := hex.DecodeString(seedHex)
	assert.NoError(t, err)
	w := New(seed)

	ltcAccount, err := w.CoinAccount(LTC, true, 0)
	assert.NoError(t, err)
	t.Log("coin account:", ltcAccount)

	for i := uint32(0); i < 2; i++ {
		a, err := ltcAccount.Address(i, false)
		assert.NoError(t, err)
		t.Logf("external_%d: %s", i, a)
		a, err = ltcAccount.Address(i, true)
		t.Logf("internal_%d: %s", i, a)
	}
}
