// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package address

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/util"
)

// to hold the type of the address
type Version byte

// to hold thefixed-length address bytes
type AddressBytes [20]byte

// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
const (
	BtcLivenet       Version = 0
	BtcLivenetScript Version = 5
	BtcTestnet       Version = 111
	BtcTestnetScript Version = 196

	LtcLivenet Version = 48
	//LtcLivenetScript  Version = 5
	LtcLivenetScript2 Version = 50
	//LtcTestnet        Version = 111
	//LtcTestnetScript  Version = 196
	LtcTestnetScript2 Version = 58

	vNull Version = 0xff

	expectedAddressLenght = 25
)

// check the address and return its version
func ValidateAddress(address string) (Version, AddressBytes, error) {

	addr := util.FromBase58(address)
	addressBytes := AddressBytes{}

	if expectedAddressLenght != len(addr) {
		return vNull, addressBytes, fmt.Errorf("address bytes length: %d expected: %d", len(addr), expectedAddressLenght)
	}

	h := sha256.New()
	h.Write(addr[:21])
	d := h.Sum([]byte{})
	h = sha256.New()
	h.Write(d)
	d = h.Sum([]byte{})

	if !bytes.Equal(d[0:4], addr[21:]) {
		return vNull, addressBytes, fmt.Errorf("address checksum failed: %x expected: %s", d[0:4], addr[21:])
	}

	version := Version(addr[0])
	switch version {
	case BtcLivenet, BtcLivenetScript, BtcTestnet, BtcTestnetScript:
	case LtcLivenet, LtcLivenetScript2, LtcTestnetScript2:
	default:
		return vNull, addressBytes, fmt.Errorf("address version: %d is invalid", int(addr[0]))
	}

	copy(addressBytes[:], addr[1:21])

	return version, addressBytes, nil
}
