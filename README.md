# sampleStress

0. open main.test.go
```golang
var mnemonic = "pepper hair process town say voyage exhibit over carry property follow define"
// Rich account
var accountInit, _ = FromHexKey("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
var accountCount = 200 // account and Number of threads
```
1. TestInitAccount     Initial balance
2. TestQueryAccountsBalance
3. TestBatchTransactions
