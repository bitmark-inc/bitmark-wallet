package wallet

import (
	"github.com/bitgoin/address"
)

type CoinType string
type Test bool

const (
	BTC CoinType = "BTC"
	LTC CoinType = "LTC"
)

var CoinMap = map[CoinType]uint32{
	BTC: 0,
	LTC: 2,
}

var CoinFee = map[CoinType]uint64{
	BTC: 20000,
	LTC: 200000,
}

var (
	//BitcoinMain is params for main net.
	BitcoinMain = &address.Params{
		DumpedPrivateKeyHeader: []byte{128},
		AddressHeader:          []byte{0},
		P2SHHeader:             []byte{5},
		HDPrivateKeyID:         []byte{0x04, 0x88, 0xad, 0xe4},
		HDPublicKeyID:          []byte{0x04, 0x88, 0xb2, 0x1e},
	}
	//BitcoinTest is params for test net.
	BitcoinTest = &address.Params{
		DumpedPrivateKeyHeader: []byte{239},
		AddressHeader:          []byte{111},
		P2SHHeader:             []byte{196},
		HDPrivateKeyID:         []byte{0x04, 0x35, 0x83, 0x94},
		HDPublicKeyID:          []byte{0x04, 0x35, 0x87, 0xcf},
	}
	//LitecoinMain is params for litecoin main net.
	LitecoinMain = &address.Params{
		DumpedPrivateKeyHeader: []byte{176},
		AddressHeader:          []byte{48},
		P2SHHeader:             []byte{50},
		HDPrivateKeyID:         []byte{0x04, 0x88, 0xad, 0xe4},
		HDPublicKeyID:          []byte{0x04, 0x88, 0xb2, 0x1e},
	}
	//LitecoinTest is params for litecoin test net.
	LitecoinTest = &address.Params{
		DumpedPrivateKeyHeader: []byte{239},
		AddressHeader:          []byte{111},
		P2SHHeader:             []byte{196},
		HDPrivateKeyID:         []byte{0x04, 0x35, 0x83, 0x94},
		HDPublicKeyID:          []byte{0x04, 0x35, 0x87, 0xcf},
	}
)

var CoinParams = map[CoinType]map[Test]*address.Params{
	BTC: {
		true:  BitcoinTest,
		false: BitcoinMain,
	},
	LTC: {
		true:  LitecoinTest,
		false: LitecoinMain,
	},
}
