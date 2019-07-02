package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDaemonGetAddr(t *testing.T) {
	d := NewDaemonAgent("http://localhost:17001/", "btcuser1",
		"pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw")
	err := d.WatchAddress("mvxpcRGnjRpme59CAnLHTxFjwd8ivwWbQb")
	assert.NoError(t, err)
	err = d.WatchAddress("1DURpDjr49tUbbMhQsG1jeAA6dq5Z5fF3p")
	assert.EqualError(t, err, "no transaction for the address")
}
