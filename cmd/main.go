package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/NebulousLabs/entropy-mnemonics"
	"github.com/bitmark-inc/bitmark-wallet"
	"github.com/boltdb/bolt"
	// "github.com/hashicorp/hcl"

	"github.com/spf13/cobra"
	"path"
)

var (
	ErrConfigBucketNotFound = fmt.Errorf("config bucket is not found")
)

func returnIfErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func genSeed(seedLen int) ([]byte, error) {
	b := make([]byte, seedLen)
	_, err := rand.Read(b)
	return b, err
}

func encryptSeed(seed, passHash []byte) ([]byte, error) {
	block, err := aes.NewCipher(passHash)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(seed))
	iv := ciphertext[:aes.BlockSize]

	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], seed)

	return ciphertext, nil
}

func decryptSeed(encryptSeed, passHash []byte) ([]byte, error) {
	block, err := aes.NewCipher(passHash)
	if err != nil {
		return nil, err
	}
	if len(encryptSeed) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := encryptSeed[:aes.BlockSize]
	encryptedText := encryptSeed[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(encryptedText, encryptedText)

	return encryptedText, nil
}

func dblSHA256(b []byte) []byte {
	h := sha256.Sum256([]byte(b))
	hash := sha256.Sum256(h[:])
	return hash[:]
}

func setConfig(dataFile string, key, value []byte) error {
	db, err := bolt.Open(dataFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			return err
		}
		return bkt.Put(key, value)
	})
}

func getConfig(dataFile string, key []byte) ([]byte, error) {
	db, err := bolt.Open(dataFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	b := make([]byte, 0)
	if err := db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("config"))
		if bkt == nil {
			return ErrConfigBucketNotFound
		}
		b = append(b, bkt.Get(key)...)
		return nil
	}); err != nil {
		return nil, err
	}
	return b, nil
}

func main() {
	var datadir, conf string
	var rootCmd = &cobra.Command{
		Use:   "bitmark-wallet",
		Short: "bitmark-wallet is a wallet supports multiple crypto currencies",
		Long:  `bitmark-wallet is a wallet supports multiple crypto currencies`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVarP(&datadir, "datadir", "d", ".", "Directory for the wallet data")
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "F", "wallet.dat", "Filename of wallet data")
	rootCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "init a wallet",
		Long:  `init a wallet`,
		Run: func(cmd *cobra.Command, args []string) {
			seed, err := genSeed(32)

			if err != nil {
				log.Fatal(err)
			}
			// fmt.Println("Seed:", hex.EncodeToString(seed))

			password, err := readPassword("Set wallet password (length >= 8): ", 8)
			if err != nil {
				log.Fatal(err)
			}

			passHash := dblSHA256([]byte(password))
			fmt.Println("SEED:", hex.EncodeToString(passHash))

			// use key to encrypt seed
			encryptedSeed, err := encryptSeed(seed, passHash[:])
			if nil != err {
				log.Fatal(err)
			}

			confFile := path.Join(datadir, conf)
			os.Remove(confFile)
			err = setConfig(confFile, []byte("HASH"), dblSHA256(seed))
			err = setConfig(confFile, []byte("SEED"), encryptedSeed)
			if nil != err {
				log.Fatal(err)
			}

			// fmt.Println("Encrypted seed:", hex.EncodeToString(encryptedSeed))

			phrase, err := mnemonics.ToPhrase(encryptedSeed, mnemonics.English)
			if err != nil {
				log.Fatal(err)
			}
			// return mnemonic phrases
			fmt.Println("Please write down the mnemonic phrases for wallet recovery:")
			fmt.Println(phrase)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "restore",
		Short: "restore a wallet from the mnemonic phrases",
		Long:  `recover a wallet from the mnemonic phrases`,
		Run: func(cmd *cobra.Command, args []string) {
			phrases, err := readMnemonic()
			if err != nil {
				log.Fatal(err)
			}

			encryptedSeed, err := mnemonics.FromString(phrases, mnemonics.English)
			// fmt.Println("Encrypted Seed:", hex.EncodeToString(encryptedSeed))

			password, err := readPassword("Set wallet password (length >= 8): ", 8)
			if err != nil {
				log.Fatal(err)
			}
			passHash := dblSHA256([]byte(password))

			seed, err := decryptSeed(append([]byte{}, encryptedSeed...), passHash[:])

			confFile := path.Join(datadir, conf)
			os.Remove(confFile)

			err = setConfig(confFile, []byte("HASH"), dblSHA256(seed))
			if nil != err {
				log.Fatal(err)
			}
			err = setConfig(confFile, []byte("SEED"), encryptedSeed)
			if nil != err {
				log.Fatal(err)
			}

			// fmt.Println("Seed:", hex.EncodeToString(seed))
		},
	})

	rootCmd.AddCommand(NewCoinCmd("btc", "Bitcoin wallet", "Bitcoin wallet", wallet.BTC))
	rootCmd.AddCommand(NewCoinCmd("ltc", "Litecoin wallet", "Litecoin wallet", wallet.LTC))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
