# tx

Cut down version of: github.com/bitgoin/tx
See that repo for further info

This has the following changes

* P2SH functions have been removed as they are not used by this program
* P2PK now call a better base58decoder
    + the bigoin/address/DecodeAddress is broken and just drops the version byte
    + this call was replace by function correctly decodes Bitcoin/Litecoin address versions
    + now unsupported address will cause error rather than the original
      generation of nonredeemable TXO
* P2PK will work with scriptPubKey.type of "scripthash" and "pubkeyhash"
