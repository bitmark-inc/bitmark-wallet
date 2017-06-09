package agent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBlockrGetAddr(t *testing.T) {
	d := NewBlockrAgent("btc.blockr.io")
	utxos, err := d.GetAddrUnspent("198aMn6ZYAczwrE5NvNTUMyJ5qkfy4g3Hi")
	assert.NoError(t, err)
	assert.NotNil(t, utxos)
	utxos, err = d.GetAddrUnspent("1DURpDjr49tUbbMhQsG1jeAA6dq5Z5fF3p")
	assert.EqualError(t, err, "no transaction for the address")
}

func TestBlockrSendError(t *testing.T) {
	d := NewBlockrAgent("btc.blockr.io")
	txId, err := d.Send("0100000002f0f51014c80e31e85372a65287a8a10ef7597a5c5d8c303cfa0d7589cdff1f34000000006b483045022100e5c22cce7c638589fba11d25f1f3f62b9fb4751f5d56eba3b7f099decdd2d008022009298f46da79d14b7967a2e0617db853b0b76d754387ea523da06382970346020121029dfce75cfc34ec6743ca16cf0d4f0ce60cc38e850dd543511d141aadf1b6fff4ffffffffb2aa3e959200d93feb4d0f571e108e58491a957bbe4e3bf30348fb6cde92691a000000006b483045022100eeb25c1f4126b53feca314d92947b8ff7124892d89380bea4897f8bd516a34b902207619084f868644ac8852fc1e5d9823ba18cf15b9a5fddca53a5297da206e037d0121029dfce75cfc34ec6743ca16cf0d4f0ce60cc38e850dd543511d141aadf1b6fff4ffffffff02b9d1fb78000000001976a914aaa4a49b08907883696bdced0ea84c320d7a6cd988ac005ed0b2000000001976a9147c154ed1dc59609e3d26abb2df2ea3d587cd8c4188ac00000000")
	assert.Error(t, err, "Could not push your transaction! (Did you sign your transaction?)")
	assert.Equal(t, "", txId)
}
