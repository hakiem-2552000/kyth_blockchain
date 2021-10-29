package main

import (
	"golang_blockchain/blockchain"
	"os"
)

func main() {
	defer os.Exit(0)
	chain := blockchain.InitBlockchain("ad")
	defer chain.Database.Close()

	cli := CommandLine{chain}
	cli.run()
}
