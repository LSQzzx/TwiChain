package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"twichain/internal/crypto"
	"twichain/internal/storage"
)

type Blockchain struct {
    Chain                    []*Block             `json:"chain"`
    CurrentTransactions      []Transaction        `json:"current_transactions"`
    Nodes                    map[string]bool      `json:"nodes"`
    mu                       sync.RWMutex         `json:"-"`
    storage                  storage.BlockStorage `json:"-"`
    Difficulty              int                  `json:"difficulty"`
    port                    string               `json:"-"` // 添加端口字段
}

// GetChain 返回区块链的副本
func (bc *Blockchain) GetChain() []*Block {
	bc.mu.RLock()
	chainCopy := make([]*Block, len(bc.Chain))
	copy(chainCopy, bc.Chain)
	bc.mu.RUnlock()
	return chainCopy
}

// GetChainLength 返回区块链长度
func (bc *Blockchain) GetChainLength() int {
	bc.mu.RLock()
	length := len(bc.Chain)
	bc.mu.RUnlock()
	return length
}

func NewBlockchain(store storage.BlockStorage, nodeAddress string, port string) *Blockchain {
    log.Printf("Initializing new blockchain on port %s", port)

    bc := &Blockchain{
        Chain:               make([]*Block, 0),
        CurrentTransactions: make([]Transaction, 0),
        Nodes:              make(map[string]bool),
        storage:            store,
        Difficulty:         2,
        port:              port,
    }

	// 如果配置了节点地址,从该节点同步数据
	if nodeAddress != "" {
		if err := bc.syncFromNode(nodeAddress); err != nil {
			log.Printf("Failed to sync from node %s: %v", nodeAddress, err)
			return nil
		}
	} else {
		// 创建创世块
		genesisTransaction := Transaction{
			ID:        generateTransactionID(),
			Sender:    "SYSTEM",
			Receiver:  "69c5f684026e6bd3e2a8f175a892ca6858cb9936b3c525ce11b981f848a69fc2",
			Signature: "GENESIS", // 创世块不需要签名验证
			IsLike:    false,
			Message:   "Genesis Block - Social Blockchain Initialized",
			Timestamp: time.Now(),
		}

		bc.CurrentTransactions = append(bc.CurrentTransactions, genesisTransaction)
		genesisBlock := bc.NewBlock(100, "1")
		log.Printf("Genesis block created with social transaction: %+v", genesisBlock)
	}

	// 启动定时挖矿
	bc.StartMining()
	return bc
}

func (bc *Blockchain) NewBlock(proof int64, previousHash string) *Block {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	block := &Block{
		Index:        len(bc.Chain) + 1,
		Timestamp:    time.Now(),
		Transactions: bc.CurrentTransactions, // 改为大写
		Proof:        proof,
		PrevHash:     previousHash,
	}

	// 转换为存储格式并保存
	blockData := &storage.BlockData{
		Index:        block.Index,
		Timestamp:    block.Timestamp,
		Proof:        block.Proof,
		PrevHash:     block.PrevHash,
		Transactions: make([]storage.TransactionData, len(block.Transactions)),
	}

	for i, tx := range block.Transactions {
		blockData.Transactions[i] = storage.TransactionData{
			ID:        tx.ID,
			Sender:    tx.Sender,
			Receiver:  tx.Receiver,
			Signature: tx.Signature,
			IsLike:    tx.IsLike,
			Timestamp: tx.Timestamp,
			Message:   tx.Message,
		}
	}

	if err := bc.storage.SaveBlock(blockData); err != nil {
		log.Printf("Error saving block: %v", err)
	}

	// 重置当前交易
	bc.CurrentTransactions = make([]Transaction, 0) // 改为大写
	bc.Chain = append(bc.Chain, block)
	return block
}

// 用于生成交易ID
func generateTransactionID() string {
	return crypto.Hash([]byte(time.Now().String()))
}

func (bc *Blockchain) ProofOfWork(lastBlock *Block) int64 {
	lastProof := lastBlock.Proof
	lastHash := crypto.HashBlock(lastBlock)

	var proof int64 = 0
	for !bc.ValidProof(lastProof, proof, lastHash) {
		proof++
	}

	return proof
}

// ValidProof 验证工作量证明
func (bc *Blockchain) ValidProof(lastProof, proof int64, lastHash string) bool {
	guess := []byte(strconv.FormatInt(lastProof, 10) + strconv.FormatInt(proof, 10) + lastHash)
	guessHash := crypto.Hash(guess)
	zeros := strings.Repeat("0", bc.Difficulty)
	return guessHash[:bc.Difficulty] == zeros
}

// RegisterNode 注册一个新的节点到网络中
func (bc *Blockchain) RegisterNode(address string) error {
    // 1. 首先进行地址验证（不需要锁）
    parsedURL, err := url.Parse(address)
    if err != nil {
        return fmt.Errorf("invalid address format: %v", err)
    }

    if parsedURL.Host == "" {
        return fmt.Errorf("invalid address: no host found")
    }

    // 2. 检查节点是否存在（需要读锁）
    bc.mu.RLock()
    exists := bc.Nodes[parsedURL.Host]
    bc.mu.RUnlock()
    
    if exists {
        return fmt.Errorf("node already exists: %s", parsedURL.Host)
    }

    // 3. 检查数据库（不需要锁）
    nodes, err := bc.storage.GetAllNodes()
    if err != nil {
        return fmt.Errorf("failed to check existing nodes: %v", err)
    }
    for _, node := range nodes {
        if node == parsedURL.Host {
            return fmt.Errorf("node already exists in database: %s", parsedURL.Host)
        }
    }

    // 4. 保存节点（需要写锁）
    bc.mu.Lock()
    if err := bc.storage.SaveNode(parsedURL.Host); err != nil {
        bc.mu.Unlock()
        return fmt.Errorf("failed to save node: %v", err)
    }
    bc.Nodes[parsedURL.Host] = true
    bc.mu.Unlock()

    // 5. 广播新节点（不需要锁）
    go bc.BroadcastNewNode(address)  // 异步执行广播

    return nil
}

// 添加节点删除方法
func (bc *Blockchain) removeNode(address string) {
	if err := bc.storage.DeleteNode(address); err != nil {
		log.Printf("Failed to delete node from storage: %v", err)
	}
	delete(bc.Nodes, address)
}

// 广播新节点
func (bc *Blockchain) BroadcastNewNode(newNode string) {
	nodes, err := bc.storage.GetAllNodes()
	if err != nil {
		log.Printf("Failed to get nodes: %v", err)
		return
	}

	data := map[string]string{
		"node": newNode,
	}

	for _, node := range nodes {
		if node == newNode {
			continue
		}
		url := fmt.Sprintf("http://%s/nodes/new", node)
		jsonData, _ := json.Marshal(data)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				log.Printf("Node %s appears to be offline, removing...", node)
				bc.removeNode(node)
			}
			continue
		}
		resp.Body.Close()
	}
}

func (bc *Blockchain) NewTransaction(sender, receiver, signature string, isLike bool, message string, targetPostID string) int {
	transaction := Transaction{
		ID:           generateTransactionID(),
		Sender:       sender,
		Receiver:     receiver,
		Signature:    signature,
		IsLike:       isLike,
		Message:      message,
		Timestamp:    time.Now(),
		TargetPostID: targetPostID,
	}

	bc.mu.Lock()
	bc.CurrentTransactions = append(bc.CurrentTransactions, transaction)
	nextBlockIndex := len(bc.Chain) + 1
	bc.mu.Unlock()

	return nextBlockIndex
}

func (bc *Blockchain) Mine() {
    // 1. 检查并复制交易（使用读锁）
    bc.mu.RLock()
    if len(bc.CurrentTransactions) == 0 {
        bc.mu.RUnlock()
        return
    }
    transactions := make([]Transaction, len(bc.CurrentTransactions))
    copy(transactions, bc.CurrentTransactions)
    lastBlock := bc.Chain[len(bc.Chain)-1]
    bc.mu.RUnlock()

    // 2. 进行工作量证明计算（不需要锁）
    proof := bc.ProofOfWork(lastBlock)
    lastHash := crypto.HashBlock(lastBlock)

    // 3. 创建新区块
    block := &Block{
        Index:        lastBlock.Index + 1,
        Timestamp:    time.Now(),
        Transactions: transactions,
        Proof:        proof,
        PrevHash:     lastHash,
    }

    // 4. 保存区块（使用写锁）
    bc.mu.Lock()
    // 再次检查条件
    if block.Index != bc.Chain[len(bc.Chain)-1].Index+1 {
        bc.mu.Unlock()
        return
    }

    // 保存区块数据
	blockData := &storage.BlockData{
		Index:        block.Index,
		Timestamp:    block.Timestamp,
		Proof:        block.Proof,
		PrevHash:     block.PrevHash,
		Transactions: make([]storage.TransactionData, len(block.Transactions)),
	}

	// 转换交易数据
	for i, tx := range block.Transactions {
		blockData.Transactions[i] = storage.TransactionData{
			ID:           tx.ID,
			Sender:       tx.Sender,
			Receiver:     tx.Receiver,
			Signature:    tx.Signature,
			IsLike:       tx.IsLike,
			Timestamp:    tx.Timestamp,
			Message:      tx.Message,
			TargetPostID: tx.TargetPostID,
		}
	}

	if err := bc.storage.SaveBlock(blockData); err != nil {
		log.Printf("Error saving block: %v", err)
		bc.mu.Unlock()
		return
	}

    // 更新内存状态
    bc.Chain = append(bc.Chain, block)
    bc.CurrentTransactions = bc.CurrentTransactions[len(transactions):]
    bc.mu.Unlock()

    // 5. 广播新区块（不需要锁）
    go bc.AnnounceNewBlock(block)  // 异步执行广播
}

func (bc *Blockchain) StartMining() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			bc.mu.RLock()
			if len(bc.CurrentTransactions) == 0 {
				bc.mu.RUnlock()
				continue
			}
			bc.mu.RUnlock()
			bc.Mine()
		}
	}()
}

func (bc *Blockchain) AnnounceNewBlock(block *Block) {
	// fmt.Printf("\nAnnouncing new block on {%s}\n", bc.port)
	nodes, err := bc.storage.GetAllNodes()
	if err != nil {
		log.Printf("Failed to get nodes: %v", err)
		return
	}

	blockData := map[string]interface{}{
		"index":         block.Index,
		"transactions":  block.Transactions,
		"timestamp":     block.Timestamp,
		"proof":         block.Proof,
		"previous_hash": block.PrevHash,
	}

	for _, node := range nodes {
		url := fmt.Sprintf("http://%s/block/receive", node)
		jsonData, _ := json.Marshal(blockData)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				log.Printf("Node %s appears to be offline, removing...", node)
				bc.removeNode(node)
			}
			continue
		}
		resp.Body.Close()
	}
}

// AddBlock 添加区块到链中
func (bc *Blockchain) AddBlock(block *Block) error {
	// fmt.Printf("\nAdding block on {%s}\n", bc.port)
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 验证区块
	lastBlock := bc.Chain[len(bc.Chain)-1]
	if block.Index != lastBlock.Index+1 {
		return fmt.Errorf("invalid block index")
	}

	if block.PrevHash != crypto.HashBlock(lastBlock) {
		return fmt.Errorf("invalid previous hash")
	}

	// 验证工作量证明
	if !bc.ValidProof(lastBlock.Proof, block.Proof, block.PrevHash) {
		return fmt.Errorf("invalid proof of work")
	}

	// 验证所有交易的签名
	for _, tx := range block.Transactions {
		var messageBytes []byte
		if tx.IsLike {
			messageBytes = []byte(tx.TargetPostID)
		} else {
			messageBytes = []byte(tx.Message)
		}

		valid, err := crypto.Verify(tx.Sender, messageBytes, tx.Signature)
		if err != nil || !valid {
			return fmt.Errorf("invalid transaction signature: %v", err)
		}
	}

	// 转换为存储格式并保存
	blockData := &storage.BlockData{
		Index:        block.Index,
		Timestamp:    block.Timestamp,
		Proof:        block.Proof,
		PrevHash:     block.PrevHash,
		Transactions: make([]storage.TransactionData, len(block.Transactions)),
	}

	// 转换交易数据
	for i, tx := range block.Transactions {
		blockData.Transactions[i] = storage.TransactionData{
			ID:           tx.ID,
			Sender:       tx.Sender,
			Receiver:     tx.Receiver,
			Signature:    tx.Signature,
			IsLike:       tx.IsLike,
			Timestamp:    tx.Timestamp,
			Message:      tx.Message,
			TargetPostID: tx.TargetPostID,
		}
	}

	// 保存到存储
	if err := bc.storage.SaveBlock(blockData); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}

	// 添加到链中
	bc.Chain = append(bc.Chain, block)

	// 清理当前交易池中已经被打包的交易
	// bc.CurrentTransactions = make([]Transaction, 0)

	return nil
}

// 同步区块链数据
func (bc *Blockchain) syncFromNode(nodeAddress string) error {
	// 创建请求数据
    data := map[string]string{
        "node": fmt.Sprintf("http://localhost:%s", bc.port),  // 需要在 Blockchain 结构体中添加 port 字段
    }
    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal request data: %v", err)
    }

    // 发送 POST 请求
    resp, err := http.Post(
        fmt.Sprintf("http://%s/nodes/register", nodeAddress),
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        log.Printf("Failed to sync from node %s: %v", nodeAddress, err)
        return err
    }
    defer resp.Body.Close()

    // 读取响应内容进行调试
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response body: %v", err)
    }
    log.Printf("Response from node: %s", string(body))

    // 解码响应
    var result struct {
        Chain []*Block        `json:"chain"`
        Nodes map[string]bool `json:"nodes"`
    }

    if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
        log.Printf("Failed to decode response from node %s: %v", nodeAddress, err)
        log.Printf("Response body: %s", string(body))
        return err
    }

	// 保存链和节点信息到内存
	bc.Chain = result.Chain
	bc.Nodes = result.Nodes
	bc.Nodes[nodeAddress] = true

	// 保存区块到存储
	for _, block := range result.Chain {
		blockData := &storage.BlockData{
			Index:        block.Index,
			Timestamp:    block.Timestamp,
			Proof:        block.Proof,
			PrevHash:     block.PrevHash,
			Transactions: make([]storage.TransactionData, len(block.Transactions)),
		}
		// 转换交易数据
		for i, tx := range block.Transactions {
			blockData.Transactions[i] = storage.TransactionData{
				ID:           tx.ID,
				Sender:       tx.Sender,
				Receiver:     tx.Receiver,
				Signature:    tx.Signature,
				IsLike:       tx.IsLike,
				Timestamp:    tx.Timestamp,
				Message:      tx.Message,
				TargetPostID: tx.TargetPostID,
			}
		}
		if err := bc.storage.SaveBlock(blockData); err != nil {
			return fmt.Errorf("failed to save block: %v", err)
		}
	}

	// 保存节点信息到数据库
	for nodeAddr := range result.Nodes {
		if err := bc.storage.SaveNode(nodeAddr); err != nil {
			log.Printf("Warning: Failed to save node %s: %v", nodeAddr, err)
			// 继续处理其他节点，不中断同步过程
		}
	}

	log.Printf("Successfully synced %d blocks and %d nodes from %s",
		len(result.Chain), len(result.Nodes), nodeAddress)
	return nil
}
