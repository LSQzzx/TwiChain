package storage

import (
	"time"
)

// BlockData 定义区块数据结构
type BlockData struct {
	Index        int               `json:"index"`
	Timestamp    time.Time         `json:"timestamp"`
	Proof        int64             `json:"proof"`
	PrevHash     string            `json:"previous_hash"`
	Transactions []TransactionData `json:"transactions"`
}

// TransactionData 定义交易数据结构
type TransactionData struct {
	ID           string    `json:"id"`
	Sender       string    `json:"sender"`
	Receiver     string    `json:"receiver"`
	Signature    string    `json:"signature"`
	IsLike       bool      `json:"is_like"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message"`
	TargetPostID string    `json:"target_post_id"`
}

// BlockStorage 定义区块链存储接口
type BlockStorage interface {
	// SaveBlock 保存区块到存储
	SaveBlock(block *BlockData) error

	// GetAllBlocks 获取所有区块
	GetAllBlocks() ([]*BlockData, error)

	// GetBlockByIndex 根据索引获取区块
	GetBlockByIndex(index int) (*BlockData, error)

	// GetBlockByHash 根据哈希获取区块
	GetBlockByHash(hash string) (*BlockData, error)

	// GetTransactionsByBlockIndex 获取指定区块的所有交易
	GetTransactionsByBlockIndex(blockIndex int) ([]TransactionData, error)

	// Close 关闭存储连接
	Close() error

	// 节点管理相关方法
	SaveNode(address string) error
	GetAllNodes() ([]string, error)
	DeleteNode(address string) error
}
