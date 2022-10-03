package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
)

type TokenBuilder struct {
	secretKey []byte
}

func InitTokenBuilder(key string) *TokenBuilder {
	return &TokenBuilder{secretKey: []byte(key)}
}

func (tb *TokenBuilder) CreateToken(id string) (string, error) {
	uintID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return "", err
	}
	binaryID := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(binaryID, uintID)
	t := append(binaryID, tb.createHash(binaryID)...)
	return hex.EncodeToString(t), nil
}

func (tb *TokenBuilder) IsTokenValid(token string) bool {
	hash, err := hex.DecodeString(token)
	if err != nil {
		return false
	}
	return hmac.Equal(hash[binary.MaxVarintLen64:], tb.createHash(hash[:binary.MaxVarintLen64]))
}

func (tb *TokenBuilder) GetIDFromToken(token string) (string, error) {
	hash, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(binary.BigEndian.Uint64(hash[:binary.MaxVarintLen64]), 10), nil
}

func (tb *TokenBuilder) createHash(v []byte) []byte {
	h := hmac.New(sha256.New, tb.secretKey)
	h.Write(v)
	return h.Sum(nil)
}
