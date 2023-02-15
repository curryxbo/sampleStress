package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/go-bip39"
	"github.com/mantlenetworkio/mantle/l2geth/accounts/abi/bind"
	"github.com/mantlenetworkio/mantle/l2geth/common"
	"github.com/mantlenetworkio/mantle/l2geth/core/types"
	"github.com/mantlenetworkio/mantle/l2geth/crypto"
	"github.com/mantlenetworkio/mantle/l2geth/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

var mnemonic = "pepper hair process town say voyage exhibit over carry property follow define"
var accountInit, _ = FromHexKey("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
var accountCount = 200 // account and Number of threads

func FromHexKey(hexkey string) (ExtAcc, error) {
	key, err := crypto.HexToECDSA(hexkey)
	if err != nil {
		return ExtAcc{}, err
	}
	pubKey := key.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		err = fmt.Errorf("publicKey is not of type *ecdsa.PublicKey")
		return ExtAcc{}, err
	}
	addr := crypto.PubkeyToAddress(*pubKeyECDSA)
	return ExtAcc{key, addr}, nil
}

type ExtAcc struct {
	Key  *ecdsa.PrivateKey
	Addr common.Address
}

func TestInitAccount(t *testing.T) {
	txOpt := bind.NewKeyedTransactor(accountInit.Key)

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		panic(err)
	}
	//txOpt.Value = big.NewInt(5e18)
	txOpt.GasLimit = uint64(21000)
	txOpt.GasPrice = big.NewInt(1)
	seed := bip39.NewSeed(mnemonic, "")
	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		log.Fatal(err)
	}
	balance, _ := client.BalanceAt(context.Background(), accountInit.Addr, nil)
	sendValue := balance.Div(balance, big.NewInt(int64(accountCount*2)))
	for i := 0; i < accountCount; i++ {
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		receiveAccount := common.HexToAddress(account.Address.Hex())
		txOpt.Value = sendValue
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1)
		nonce, err := client.PendingNonceAt(context.Background(), accountInit.Addr)
		if err != nil {
			panic(err)
		}
		txOpt.Value = sendValue
		rawTx := types.NewTransaction(nonce, receiveAccount, txOpt.Value, txOpt.GasLimit, txOpt.GasPrice, nil)

		signedTx, err := txOpt.Signer(types.HomesteadSigner{}, txOpt.From, rawTx)
		if err != nil {
			panic(err)
		}
		err = client.SendTransaction(context.Background(), signedTx)
		fmt.Println("sendTxHash", signedTx.Hash().String())
		time.Sleep(100 * time.Millisecond)
	}
}

func TestQueryAccountsBalance(t *testing.T) {
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		panic(err)
	}
	seed := bip39.NewSeed(mnemonic, "")
	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < accountCount; i++ {
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		balance, _ := client.BalanceAt(context.Background(), common.HexToAddress(account.Address.Hex()), nil)
		fmt.Printf("index%v:,balance:%v\n", i, balance)
	}
}

func TestBatchTransactions(t *testing.T) {
	s := sync.WaitGroup{}
	s.Add(accountCount)
	seed := bip39.NewSeed(mnemonic, "")
	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		log.Fatal(err)
	}
	receiveAccount := accountInit.Addr
	for i := 0; i < accountCount; i++ {
		client, err := ethclient.Dial("http://localhost:8545")
		if err != nil {
			panic(err)
		}
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		privKey, _ := wallet.PrivateKey(account)
		txOpt := bind.NewKeyedTransactor(privKey)
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1)
		go func() {
			for i := 0; i < 10000; i++ {
				nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(account.Address.Hex()))
				if err != nil {
					panic(err)
				}
				txOpt.Value = big.NewInt(1)
				rawTx := types.NewTransaction(nonce, receiveAccount, txOpt.Value, txOpt.GasLimit, txOpt.GasPrice, nil)

				signedTx, err := txOpt.Signer(types.HomesteadSigner{}, txOpt.From, rawTx)
				if err != nil {
					panic(err)
				}
				err = client.SendTransaction(context.Background(), signedTx)
				fmt.Println("sendTxHash", signedTx.Hash().String())
				time.Sleep(100 * time.Millisecond)
			}
			s.Done()
		}()
	}
	s.Wait()
}

func TestPrintPrKey(t *testing.T) {
	seed := bip39.NewSeed(mnemonic, "")
	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < accountCount; i++ {
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		address := account.Address.Hex()
		privateKey, _ := wallet.PrivateKeyHex(account)
		fmt.Println(fmt.Sprintf("address%v:", i), address)
		fmt.Println(fmt.Sprintf("privateKey%v:", i), privateKey)
	}
}
