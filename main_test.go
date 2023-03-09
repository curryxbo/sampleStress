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
	//"github.com/mantlenetworkio/mantle/l2geth/common"
	//"github.com/mantlenetworkio/mantle/l2geth/core/types"
	//"github.com/mantlenetworkio/mantle/l2geth/crypto"
	//"github.com/mantlenetworkio/mantle/l2geth/ethclient"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

var mnemonic = "pepper hair process town say voyage exhibit over carry property follow define"
var accountInit, _ = FromHexKey("bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7")

// bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7  //bedrock
// ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
var accountCount = 500 // account and Number of threads
var txCount = 1000
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
	txOpt := bind.NewKeyedTransactor(accountInit.Key)

	client, err := ethclient.Dial("http://localhost:9545")
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
		// 创建交易对象
		//tx := types.NewTx(&types.DynamicFeeTx{
		//	ChainID:   big.NewInt(905), // 主网的 Chain ID 为 1
		//	Nonce:     nonce,           // 发送方账户的交易 nonce
		//	GasTipCap: big.NewInt(10),  // 打包人希望获得的小费，单位为 wei
		//	//MaxFeePerGas: big.NewInt(100000000000), // gas 费用上限，单位为 wei
		//	GasFeeCap: big.NewInt(50000000000), // 基础 gas 费用，单位为 wei
		//	To:        &receiveAccount,
		//	Value:     txOpt.Value, // 要发送的以太币数量，单位为 wei
		//	Data:      []byte{},
		//})
		gasPrice, _ := client.SuggestGasPrice(context.Background())

		baseTx := &types.LegacyTx{
			To:       &receiveAccount,
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      50000000000,
			Value:    txOpt.Value,
			Data:     nil,
		}
		tx := types.NewTx(baseTx)

		// 使用私钥对交易进行签名
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(tx.ChainId()), accountInit.Key)
		if err != nil {
			log.Fatal(err)
		}

		err = client.SendTransaction(context.Background(), signedTx)
		fmt.Println("sendTxHash", signedTx.Hash().String())
		time.Sleep(500 * time.Millisecond)
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
		txOpt := bind.NewKeyedTransactor(privKey)
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1)
		go func() {
			nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(account.Address.Hex()))
			if err != nil {
				fmt.Printf("PendingNonceAt error:%+v\n", err)
				s.Done()
				return
			}
			for in := 0; in < txCount; in++ {
				txOpt.Value = big.NewInt(1)
				// 创建交易对象
				//tx := types.NewTx(&types.DynamicFeeTx{
				//	ChainID:   big.NewInt(1),      // 主网的 Chain ID 为 1
				//	Nonce:     nonce + uint64(in), // 发送方账户的交易 nonce
				//	GasTipCap: big.NewInt(10),     // 打包人希望获得的小费，单位为 wei
				//	//MaxFeePerGas: big.NewInt(100000000000), // gas 费用上限，单位为 wei
				//	GasFeeCap: big.NewInt(50000000000), // 基础 gas 费用，单位为 wei
				//	To:        &receiveAccount,
				//	Value:     txOpt.Value, // 要发送的以太币数量，单位为 wei
				//	Data:      []byte{},
				//})

				gasPrice, _ := client.SuggestGasPrice(context.Background())

				baseTx := &types.LegacyTx{
					To:       &receiveAccount,
					Nonce:    nonce,
					GasPrice: gasPrice,
					Gas:      50000000000,
					Value:    txOpt.Value,
					Data:     nil,
				}
				tx := types.NewTx(baseTx)

				// 使用私钥对交易进行签名
				signedTx, err := types.SignTx(tx, types.NewEIP155Signer(tx.ChainId()), privKey)
				if err != nil {
					log.Fatal(err)
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

				time.Sleep(50 * time.Millisecond)
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
		txOpt := bind.NewKeyedTransactor(privKey)
		txOpt.GasLimit = uint64(21000)
		txOpt.GasPrice = big.NewInt(1)
		go func() {
			nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(account.Address.Hex()))
			if err != nil {
				fmt.Printf("PendingNonceAt error:%+v\n", err)
				s.Done()
				return
			}
			for in := 0; in < txCount; in++ {
				txOpt.Value = big.NewInt(1)
				// 创建交易对象
				//tx := types.NewTx(&types.DynamicFeeTx{
				//	ChainID:   big.NewInt(1),      // 主网的 Chain ID 为 1
				//	Nonce:     nonce + uint64(in), // 发送方账户的交易 nonce
				//	GasTipCap: big.NewInt(10),     // 打包人希望获得的小费，单位为 wei
				//	//MaxFeePerGas: big.NewInt(100000000000), // gas 费用上限，单位为 wei
				//	GasFeeCap: big.NewInt(50000000000), // 基础 gas 费用，单位为 wei
				//	To:        &receiveAccount,
				//	Value:     txOpt.Value, // 要发送的以太币数量，单位为 wei
				//	Data:      []byte{},
				//})

				gasPrice, _ := client.SuggestGasPrice(context.Background())

				baseTx := &types.LegacyTx{
					To:       &receiveAccount,
					Nonce:    nonce,
					GasPrice: gasPrice,
					Gas:      50000000000,
					Value:    txOpt.Value,
					Data:     nil,
				}
				tx := types.NewTx(baseTx)

				// 使用私钥对交易进行签名
				signedTx, err := types.SignTx(tx, types.NewEIP155Signer(tx.ChainId()), privKey)
				if err != nil {
					log.Fatal(err)
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
				time.Sleep(50 * time.Millisecond)
			}
			s.Done()
		}()
	}
	s.Wait()
	fmt.Println("start:", start)
	fmt.Println("end:", time.Now())
	ii := 0
	for {
		time.Sleep(1 * time.Second)
		fmt.Println("second:", ii)
		ii++
	}
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
