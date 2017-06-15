# bitmark-wallet CLI

This is the command line interface for bitmark-wallet.

## Install

```
$ go install bitmark-inc/bitmark-wallet/cmd
```

## Usage

#### Create a new wallet
```
$ bitmark-wallet init
Set wallet password (length >= 8):
SEED: 550a1539820cd545b81e4bb984fe2f4233b97327f64b39b4dcc0aec6fe3f15c0
Please write down the mnemonic phrases for wallet recovery:
vein vivid igloo hitched vogue nodes toenail identity wobbly vane fazed baby bobsled fazed duration having ticket nudged oasis womanly joking diode possible tumbling judge opacity pimple girth rigid duplex bays cousin earth icing ankle dedicated

```

#### Restore a wallet from mnemonic phrases:
```
$ bitmark-wallet restore                                                                                12:31:54 06/15/2017
IMPORTANT: all the data in the existance wallet will be removed.
Enter the mnemonic phrases for a wallet: vein vivid igloo hitched vogue nodes toenail identity wobbly vane fazed baby bobsled fazed duration having ticket nudged oasis womanly joking diode possible tumbling judge opacity pimple girth rigid duplex bays cousin earth icing ankle dedicated
Set wallet password (length >= 8):

```

### Wallet operations
```
$ bitmark-wallet ltc -t -N localhost:17001 -U btcuser -P password newaddress
Input wallet password:
Address:  mpdKBmANVPsfc98dXxhTVePjmqDHgStL7j

$ bitmark-wallet ltc -t -N localhost:17001 -U btcuser -P password sync
Input wallet password:
Sync data from network. It takes a period of time...
Balance:  67603099

$ bitmark-wallet ltc -t -N localhost:17001 -U btcuser -P password ltc -t -N localhost:17001 -U btcuser1 -P pjbgpsqvmmlmjlstkzhltwzrfgjrlsxfqzzfzshpmzstnhsdttltfmzxxkblzzcw send 'mnw1RtVwS5CRzbwV4rMxTiUqGec2DuK43n' '20000' -H 'efc02c2af662db5dcc900701fc1d77047e8432d8feabe0db949fa0967525db770a0850567d4e322e691b825632cd653d'
Input wallet password:
Fee:  28400
Raw Transaction:  5b35f3d330dbad503f2b26313b6ac0dceb7907186303ba7c7d3ab845c598e0e6

```
