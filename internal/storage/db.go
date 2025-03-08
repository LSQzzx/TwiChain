package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	connection *sql.DB
}

func NewDatabase(dataSourceName string) (BlockStorage, error) {
	// 确保数据库目录存在
	dbDir := filepath.Dir(dataSourceName)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	log.Printf("Database directory created/verified: %s", dbDir)

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// 验证连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// 创建表
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	log.Printf("Database initialized successfully at: %s", dataSourceName)
	return &Database{connection: db}, nil
}

func createTables(db *sql.DB) error {
	// 创建区块表
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS blocks (
            "index" INTEGER PRIMARY KEY,
            timestamp DATETIME,
            proof INTEGER,
            previous_hash TEXT,
            transactions TEXT
        )
    `)
	if err != nil {
		return err
	}

	// 修改交易表
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS transactions (
            id TEXT PRIMARY KEY,
            sender TEXT,
            receiver TEXT,
            signature TEXT,      -- EdDSA签名
            message TEXT,        -- 原始消息
            is_like BOOLEAN,
            timestamp DATETIME,
            target_post_id TEXT, -- 目标帖子ID
            block_index INTEGER,
            FOREIGN KEY(block_index) REFERENCES blocks("index")
        )
    `)
	if err != nil {
		return err
	}

	// 创建节点表
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS nodes (
            address TEXT PRIMARY KEY,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
	return err
}

func (db *Database) Close() error {
	return db.connection.Close()
}

// SaveBlock 修改为使用 BlockData
func (db *Database) SaveBlock(block *BlockData) error {
	tx, err := db.connection.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 序列化交易数据
	transactionsJSON, err := json.Marshal(block.Transactions)
	if err != nil {
		return err
	}

	// 插入区块
	_, err = tx.Exec(`
        INSERT INTO blocks ("index", timestamp, proof, previous_hash, transactions)
        VALUES (?, ?, ?, ?, ?)
    `, block.Index, block.Timestamp, block.Proof, block.PrevHash, string(transactionsJSON))
	if err != nil {
		return err
	}

	// 插入交易记录
	for _, transaction := range block.Transactions {
		_, err = tx.Exec(`
            INSERT INTO transactions (
                id, sender, receiver, signature, message, is_like, timestamp, target_post_id, block_index
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `, transaction.ID, transaction.Sender, transaction.Receiver, transaction.Signature,
			transaction.Message, transaction.IsLike, transaction.Timestamp,
			transaction.TargetPostID, block.Index)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetAllBlocks 修改为返回 BlockData
func (db *Database) GetAllBlocks() ([]*BlockData, error) {
	rows, err := db.connection.Query(`
        SELECT "index", timestamp, proof, previous_hash, transactions 
        FROM blocks 
        ORDER BY "index"
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []*BlockData
	for rows.Next() {
		var block BlockData
		var transactionsJSON string
		err := rows.Scan(
			&block.Index,
			&block.Timestamp,
			&block.Proof,
			&block.PrevHash,
			&transactionsJSON,
		)
		if err != nil {
			return nil, err
		}

		// 反序列化交易数据
		if err := json.Unmarshal([]byte(transactionsJSON), &block.Transactions); err != nil {
			return nil, err
		}

		blocks = append(blocks, &block)
	}

	return blocks, nil
}

// 添加新的方法实现
func (db *Database) GetBlockByIndex(index int) (*BlockData, error) {
	var block BlockData
	var transactionsJSON string

	err := db.connection.QueryRow(`
        SELECT "index", timestamp, proof, previous_hash, transactions 
        FROM blocks 
        WHERE "index" = ?
    `, index).Scan(
		&block.Index,
		&block.Timestamp,
		&block.Proof,
		&block.PrevHash,
		&transactionsJSON,
	)

	if err != nil {
		return nil, err
	}

	// 反序列化交易数据
	if err := json.Unmarshal([]byte(transactionsJSON), &block.Transactions); err != nil {
		return nil, err
	}

	return &block, nil
}

func (db *Database) GetBlockByHash(hash string) (*BlockData, error) {
	var block BlockData
	var transactionsJSON string

	err := db.connection.QueryRow(`
        SELECT "index", timestamp, proof, previous_hash, transactions 
        FROM blocks 
        WHERE previous_hash = ?
    `, hash).Scan(
		&block.Index,
		&block.Timestamp,
		&block.Proof,
		&block.PrevHash,
		&transactionsJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %v", err)
	}

	if err := json.Unmarshal([]byte(transactionsJSON), &block.Transactions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transactions: %v", err)
	}

	return &block, nil
}

func (db *Database) GetTransactionsByBlockIndex(blockIndex int) ([]TransactionData, error) {
	rows, err := db.connection.Query(`
        SELECT id, sender, receiver, signature, is_like, timestamp, message, target_post_id
        FROM transactions 
        WHERE block_index = ?
        ORDER BY timestamp
    `, blockIndex)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []TransactionData
	for rows.Next() {
		var tx TransactionData
		if err := rows.Scan(
			&tx.ID,
			&tx.Sender,
			&tx.Receiver,
			&tx.Signature,
			&tx.IsLike,
			&tx.Timestamp,
			&tx.Message,
			&tx.TargetPostID,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// 实现节点存储方法
func (db *Database) SaveNode(address string) error {
	_, err := db.connection.Exec(`
        INSERT OR REPLACE INTO nodes (address) VALUES (?)
    `, address)
	return err
}

func (db *Database) GetAllNodes() ([]string, error) {
	rows, err := db.connection.Query(`SELECT address FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []string
	for rows.Next() {
		var address string
		if err := rows.Scan(&address); err != nil {
			return nil, err
		}
		nodes = append(nodes, address)
	}
	return nodes, nil
}

func (db *Database) DeleteNode(address string) error {
	_, err := db.connection.Exec(`DELETE FROM nodes WHERE address = ?`, address)
	return err
}
