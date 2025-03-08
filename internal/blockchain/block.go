package blockchain

import (
    "time"
)

type Block struct {
    Index        int         `json:"index"`
    Timestamp    time.Time   `json:"timestamp"`
    Transactions []Transaction `json:"transactions"`
    Proof        int64       `json:"proof"`
    PrevHash     string      `json:"previous_hash"`
}

func NewBlock(index int, transactions []Transaction, proof int64, prevHash string) *Block {
    return &Block{
        Index:        index,
        Timestamp:    time.Now(),
        Transactions: transactions,
        Proof:        proof,
        PrevHash:     prevHash,
    }
}