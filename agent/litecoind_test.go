package agent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLitecoindGetAddr(t *testing.T) {
	d := NewLitecoindAgent("http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw")
	utxos, err := d.GetAddrUnspent("mvxpcRGnjRpme59CAnLHTxFjwd8ivwWbQb")
	assert.NoError(t, err)
	assert.NotNil(t, utxos)
	utxos, err = d.GetAddrUnspent("1DURpDjr49tUbbMhQsG1jeAA6dq5Z5fF3p")
	assert.EqualError(t, err, "no transaction for the address")
}
