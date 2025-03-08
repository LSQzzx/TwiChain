package blockchain

import "time"

// Transaction 代表区块链中的一个交互行为(发帖/评论/点赞)
type Transaction struct {
	ID           string    `json:"id"`             // 交易ID
	Sender       string    `json:"sender"`         // 发送者地址(256位十六进制)
	Receiver     string    `json:"receiver"`       // 接收者地址(256位十六进制)
	Signature    string    `json:"signature"`      // EdDSA签名(r,s)
	IsLike       bool      `json:"is_like"`        // 是否是点赞
	Timestamp    time.Time `json:"timestamp"`      // 时间戳
	Message      string    `json:"message"`        // 原始消息内容
	TargetPostID string    `json:"target_post_id"` // 目标帖子ID（点赞时必填）
}

// NewTransaction 创建新交易
func NewTransaction(id, sender, receiver, signature string, isLike bool, message string, targetPostID string) *Transaction {
	return &Transaction{
		ID:           id,
		Sender:       sender,
		Receiver:     receiver,
		Signature:    signature,
		IsLike:       isLike,
		Timestamp:    time.Now(),
		Message:      message,
		TargetPostID: targetPostID,
	}
}
