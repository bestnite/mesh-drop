package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateKey 生成新的 Ed25519 密钥对
// 返回 base64 编码的私钥和公钥
func GenerateKey() (string, string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub), nil
}

// Sign 使用私钥对数据进行签名
// privKeyStr: base64 编码的私钥
// data: 要签名的数据
// 返回: base64 编码的签名
func Sign(privKeyStr string, data []byte) (string, error) {
	privKeyBytes, err := base64.StdEncoding.DecodeString(privKeyStr)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	if len(privKeyBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key length")
	}

	signature := ed25519.Sign(ed25519.PrivateKey(privKeyBytes), data)
	return base64.StdEncoding.EncodeToString(signature), nil
}

// Verify 使用公钥验证签名
// pubKeyStr: base64 编码的公钥
// data: 原始数据
// sigStr: base64 编码的签名
func Verify(pubKeyStr string, data []byte, sigStr string) (bool, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyStr)
	if err != nil {
		return false, fmt.Errorf("invalid public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key length")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		return false, fmt.Errorf("invalid signature: %w", err)
	}

	return ed25519.Verify(ed25519.PublicKey(pubKeyBytes), data, sigBytes), nil
}
