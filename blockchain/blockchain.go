package blockchain

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"

	"github.com/boltdb/bolt"
)

const (
	dbPath       = "./tmp/blocks/blockchain.db"
	blocksBucket = "blocks"
	genesisData  = "First Transaction from Genesis"
)

type Blockchain struct {
	LastHash []byte
	Database *bolt.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *bolt.DB
}

func dbExists() bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func ContinueBlockChain(address string) *Blockchain {
	if !dbExists() {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}
	var lastHash []byte

	db, err := bolt.Open(dbPath, 0600, nil)
	Handle(err)
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		lastHash = bucket.Get([]byte("lh"))
		return nil
	})
	Handle(err)

	chain := &Blockchain{LastHash: lastHash, Database: db}
	return chain
}

func InitBlockchain(address string) *Blockchain {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var lastHash []byte
	db, err := bolt.Open(dbPath, 0600, nil)

	if err != nil {
		Handle(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))

		if bucket == nil {
			fmt.Println("No existing blockchain found. Creating a new one...")
			cbtx := CoinBaseTx(address, genesisData)
			genesis := Genesis(cbtx)

			bucket, err = tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				Handle(err)
			}

			err = bucket.Put(genesis.Hash, genesis.Serialize())
			Handle(err)

			err = bucket.Put([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		} else {
			lastHash = bucket.Get([]byte("lh"))
		}
		return nil
	})

	Handle(err)

	return &Blockchain{LastHash: lastHash, Database: db}
}

func (chain *Blockchain) AddBlock(transactions []*Transaction) {
	var lashHash []byte

	err := chain.Database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		lashHash = bucket.Get([]byte("lh"))
		return nil
	})
	Handle(err)

	newBlock := CreateBlock(transactions, lashHash)

	err = chain.Database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		err = bucket.Put(newBlock.Hash, newBlock.Serialize())
		Handle(err)

		err = bucket.Put([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	Handle(err)
}

func (chain *Blockchain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}
	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))

		encodeBlock := bucket.Get(iter.CurrentHash)
		block = Deserialize(encodeBlock)
		return nil
	})
	Handle(err)
	iter.CurrentHash = block.PrevHash
	return block
}

func (chain *Blockchain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTxs
}

func (chain *Blockchain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

func (chain *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOuts
}
