# Bitmark Wallet

This a implementation of Hierarchical Deterministic wallet according to BIP32.
It currently supports bitcoin and litecoin.

## Prerequisite

- go 1.8+

## Examples

``` golang
seed := "fded5e8970380eef15f742348d28511111366ae6a55188402b16c69922006fe6"
walletData := "wallet.dat"
w := wallet.New(seed, walletData)
coinAccount, err = w.CoinAccount(wallet.BTC, wallet.Test(test), 0)
if err != nil {
    log.Fatal(err)
}
addr, err := coinAccount.NewExternalAddr()
fmt.Println(err)
```

