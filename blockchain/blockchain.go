package blockchain

import (
	"fmt"

	"github.com/boltdb/bolt"
)

const (
	dbPath       = "./tmp/blocks/blockchain.db"
	blocksBucket = "blocks"
)

type Blockchain struct {
	LastHash []byte
	Database *bolt.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *bolt.DB
}

func InitBlockchain() *Blockchain {
	var lastHash []byte

	db, err := bolt.Open(dbPath, 0600, nil)

	if err != nil {
		Handle(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))

		if bucket == nil {
			fmt.Println("No existing blockchain found. Creating a new one...")
			genesis := Genesis()

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

func (chain *Blockchain) AddBlock(data string) {
	var lashHash []byte

	err := chain.Database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		lashHash = bucket.Get([]byte("lh"))
		return nil
	})
	Handle(err)

	newBlock := CreateBlock(data, lashHash)

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
