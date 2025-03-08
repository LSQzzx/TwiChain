package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Hash 计算数据的哈希值
func Hash(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

// HashBlock 计算区块的哈希值
func HashBlock(block interface{}) string {
	blockBytes, err := json.Marshal(block)
	if err != nil {
		return ""
	}
	return Hash(blockBytes)
}
