package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	// "log"
)

// ValidateAddress 验证地址是否为有效的 256 位十六进制字符串
func ValidateAddress(address string) bool {
	if len(address) != 64 { // 256位=64个十六进制字符
		return false
	}
	_, err := hex.DecodeString(address)
	return err == nil
}

// Sign 使用私钥对消息进行签名
func Sign(privateKey string, message []byte) (string, error) {
	privBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	priv := ed25519.PrivateKey(privBytes)
	signature := ed25519.Sign(priv, message)
	return hex.EncodeToString(signature), nil
}

// Verify 验证签名
func Verify(publicKey string, message []byte, signature string) (bool, error) {
	// 打印输入参数以便调试
	// log.Printf("Verifying signature:")
	// log.Printf("Public key (hex): %s", publicKey)
	// log.Printf("Message: %s", string(message))
	// log.Printf("Signature (hex): %s", signature)

	// 解码公钥
	pubBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return false, fmt.Errorf("invalid public key: %v (len=%d)", err, len(publicKey))
	}
	// log.Printf("Decoded public key length: %d bytes", len(pubBytes))

	// 解码签名
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature: %v (len=%d)", err, len(signature))
	}
	// log.Printf("Decoded signature length: %d bytes", len(sigBytes))

	pub := ed25519.PublicKey(pubBytes)
	result := ed25519.Verify(pub, message, sigBytes)
	// log.Printf("Verification result: %v", result)

	return result, nil
}
