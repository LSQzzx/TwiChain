package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"twichain/internal/blockchain"
	"twichain/internal/crypto"
)

type Server struct {
	blockchain *blockchain.Blockchain
	port       string
	server     *http.Server
}

func NewServer(bc *blockchain.Blockchain, port string) *Server {
	s := &Server{
		blockchain: bc,
		port:       port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/transactions/new", s.handleNewTransaction)
	mux.HandleFunc("/chain", s.handleGetChain)
	mux.HandleFunc("/nodes/register", s.handleRegisterNodes)
	mux.HandleFunc("/block/receive", s.handleReceiveBlock)
	mux.HandleFunc("/nodes/new", s.handleNewNode)

	server := &http.Server{
		Addr:           ":" + s.port,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.server = server
	return s
}

func (s *Server) Start() error {
	fmt.Printf("Server starting on port %s\n", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) handleNewTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tx struct {
		Sender       string `json:"sender"`
		Receiver     string `json:"receiver"`
		Message      string `json:"message"`   // 原始消息
		Signature    string `json:"signature"` // EdDSA签名
		IsLike       bool   `json:"is_like"`
		TargetPostID string `json:"target_post_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 验证地址格式
	if !crypto.ValidateAddress(tx.Sender) || !crypto.ValidateAddress(tx.Receiver) {
		http.Error(w, "Invalid address format - must be 256-bit hex string", http.StatusBadRequest)
		return
	}

	// 如果不是点赞，则必须有消息内容
	if !tx.IsLike && tx.Message == "" {
		http.Error(w, "Message required for non-like transactions", http.StatusBadRequest)
		return
	}

	// 添加点赞验证逻辑
	if tx.IsLike {
		if tx.TargetPostID == "" {
			http.Error(w, "Target post ID is required for likes", http.StatusBadRequest)
			return
		}
		// TODO: 验证目标帖子是否存在
	}

	// 验证签名
	var messageBytes []byte
	if tx.IsLike {
		// 对于点赞交易，使用 targetPostID 作为签名内容
		if tx.TargetPostID == "" {
			http.Error(w, "Target post ID is required for likes", http.StatusBadRequest)
			return
		}
		messageBytes = []byte(tx.TargetPostID)
	} else {
		// 对于其他类型的交易，使用消息内容
		if tx.Message == "" {
			http.Error(w, "Message is required for non-like transactions", http.StatusBadRequest)
			return
		}
		messageBytes = []byte(tx.Message)
	}

	// 验证签名
	valid, err := crypto.Verify(tx.Sender, messageBytes, tx.Signature)
	// log.Printf("Signature verification: %v, err: %v", valid, err)
	if err != nil {
		http.Error(w, fmt.Sprintf("Signature verification error: %v", err), http.StatusBadRequest)
		return
	}
	if !valid {
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	// 处理交易前广播并等待确认
	index := s.blockchain.NewTransaction(
		tx.Sender,
		tx.Receiver,
		tx.Signature, // 保存签名作为 content
		tx.IsLike,
		tx.Message, // 添加原始消息
		tx.TargetPostID,
	)

	if index == 0 {
		http.Error(w, "Transaction failed to get consensus", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message": fmt.Sprintf("Transaction will be added to Block %d", index),
		"index":   index,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chain := s.blockchain.GetChain()
	length := s.blockchain.GetChainLength()

	response := map[string]interface{}{
		"chain":  chain,
		"length": length,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding chain response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleRegisterNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Node string `json:"node"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid node data", http.StatusBadRequest)
		return
	}

	if err := s.blockchain.RegisterNode(data.Node); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			http.Error(w, fmt.Sprintf("Invalid node address: %v", err), http.StatusBadRequest)
			return
		}
	}

	response := map[string]interface{}{
		"chain": s.blockchain.GetChain(),
		"nodes": s.blockchain.Nodes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleNewNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Node string `json:"node"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid node data", http.StatusBadRequest)
		return
	}

	if err := s.blockchain.RegisterNode(data.Node); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var blockData blockchain.Block
	if err := json.NewDecoder(r.Body).Decode(&blockData); err != nil {
		http.Error(w, "Invalid block data", http.StatusBadRequest)
		return
	}

	// 添加区块到链中
	if err := s.blockchain.AddBlock(&blockData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add block: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
