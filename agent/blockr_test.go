package agent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBlockrGetAddr(t *testing.T) {
	d := NewBlockrAgent()
	utxos, err := d.GetAddrUnspent("198aMn6ZYAczwrE5NvNTUMyJ5qkfy4g3Hi")
	assert.NoError(t, err)
	assert.NotNil(t, utxos)
	utxos, err = d.GetAddrUnspent("1DURpDjr49tUbbMhQsG1jeAA6dq5Z5fF3p")
	assert.EqualError(t, err, "no transaction for the address")
}
