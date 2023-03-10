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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

var mnemonic = "pepper hair process town say voyage exhibit over carry property follow define"
var accountInit, _ = FromHexKey("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
var accountCount = 5000 // account and Number of threads
var txCount = 2
var IsSync = false

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
	//txOpt := bind.NewKeyedTransactor(accountInit.Key)
	txOpt, err := bind.NewKeyedTransactorWithChainID(accountInit.Key, big.NewInt(0x385))

	client, err := ethclient.Dial("http://localhost:9545")
	if err != nil {
		panic(err)
	}
	//txOpt.Value = big.NewInt(5e18)
	txOpt.GasLimit = uint64(21000)
	txOpt.GasPrice = big.NewInt(1000000000)
	seed := bip39.NewSeed(mnemonic, "")
	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		log.Fatal(err)
	}
	balance, _ := client.BalanceAt(context.Background(), accountInit.Addr, nil)
	sendValue := balance.Div(balance, big.NewInt(int64(accountCount*2)))
	//nonce, err := client.PendingNonceAt(context.Background(), accountInit.Addr)
	for i := 0; i < accountCount; i++ {
		nonce, err := client.PendingNonceAt(context.Background(), accountInit.Addr)
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		receiveAccount := common.HexToAddress(account.Address.Hex())
		txOpt.Value = sendValue
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1000000000)
		txOpt.Value = sendValue
		rawTx := types.NewTransaction(nonce, receiveAccount, txOpt.Value, txOpt.GasLimit, txOpt.GasPrice, nil)

		//signedTx, err := txOpt.Signer(types.HomesteadSigner{}, txOpt.From, rawTx)
		signedTx, err := txOpt.Signer(txOpt.From, rawTx)
		if err != nil {
			panic(err)
		}
		//go func() {
		err = client.SendTransaction(context.Background(), signedTx)
		//}()
		fmt.Println("sendTxHash", signedTx.Hash().String())
		time.Sleep(50 * time.Millisecond)
	}
}

func TestQueryBlockTxLen(t *testing.T) {

	client, err := ethclient.Dial("http://localhost:9545")
	if err != nil {
		panic(err)
	}
	for i := 24583; i < 24600; i++ {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)+1))
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("tx length:", len(block.Transactions()))
		fmt.Println("                   block time:", block.Time())
		fmt.Println("                   block number", block.Number().Uint64())
	}

}

func TestQueryAccountsBalance(t *testing.T) {
	client, err := ethclient.Dial("http://localhost:9545")
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
	var sy sync.Mutex
	var index int
	receiveAccount := accountInit.Addr
	start := time.Now()
	for i := 0; i < accountCount/2; i++ {
		client, err := ethclient.Dial("http://localhost:9545")
		if err != nil {
			panic(err)
		}
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		privKey, _ := wallet.PrivateKey(account)
		//txOpt := bind.NewKeyedTransactor(privKey)
		txOpt, err := bind.NewKeyedTransactorWithChainID(privKey, big.NewInt(0x385))
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1000000000)
		go func() {
			for in := 0; in < txCount; in++ {
				txOpt.Value = big.NewInt(1)
				nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(account.Address.Hex()))
				if err != nil {
					fmt.Printf("PendingNonceAt error:%+v\n", err)
					s.Done()
					return
				}
				rawTx := types.NewTransaction(nonce, receiveAccount, txOpt.Value, txOpt.GasLimit, txOpt.GasPrice, nil)

				//signedTx, err := txOpt.Signer(types.HomesteadSigner{}, txOpt.From, rawTx)
				signedTx, err := txOpt.Signer(txOpt.From, rawTx)
				if err != nil {
					continue
				}
				err = client.SendTransaction(context.Background(), signedTx)
				if IsSync {
					sy.Lock()
					fmt.Printf("index:%v,sendTxHash:%v\n", index, signedTx.Hash().String())
					index++
					sy.Unlock()
				} else {
					fmt.Printf("i:%v,in:%v,sendTxHash:%v\n", i, in, signedTx.Hash().String())
				}

				time.Sleep(100 * time.Nanosecond)
			}
			s.Done()
		}()
	}

	for i := accountCount / 2; i < accountCount; i++ {
		client, err := ethclient.Dial("http://localhost:9545")
		if err != nil {
			panic(err)
		}
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%v", i))
		account, err := wallet.Derive(path, false)
		if err != nil {
			log.Fatal(err)
		}
		privKey, _ := wallet.PrivateKey(account)
		//txOpt := bind.NewKeyedTransactor(privKey)
		txOpt, err := bind.NewKeyedTransactorWithChainID(privKey, big.NewInt(0x385))
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1000000000)
		go func() {
			for in := 0; in < txCount; in++ {
				txOpt.Value = big.NewInt(1)
				nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(account.Address.Hex()))
				if err != nil {
					fmt.Printf("PendingNonceAt error:%+v\n", err)
					s.Done()
					return
				}
				rawTx := types.NewTransaction(nonce, receiveAccount, txOpt.Value, txOpt.GasLimit, txOpt.GasPrice, nil)

				//signedTx, err := txOpt.Signer(types.HomesteadSigner{}, txOpt.From, rawTx)
				signedTx, err := txOpt.Signer(txOpt.From, rawTx)
				if err != nil {
					continue
				}
				err = client.SendTransaction(context.Background(), signedTx)
				if IsSync {
					sy.Lock()
					fmt.Printf("index:%v,sendTxHash:%v\n", index, signedTx.Hash().String())
					index++
					sy.Unlock()
				} else {
					fmt.Printf("i:%v,in:%v,sendTxHash:%v\n", i, in, signedTx.Hash().String())
				}
				time.Sleep(100 * time.Nanosecond)
			}
			s.Done()
		}()
	}
	s.Wait()
	fmt.Println("start:", start)
	fmt.Println("end:", time.Now())
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
