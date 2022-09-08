package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
)

var secretKey = []byte("secret key")

func SetSecretKey(key string) {
	secretKey = []byte(key)
}

func CreateToken(id string) (string, error) {
	uintID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return "", err
	}
	binaryID := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(binaryID, uintID)
	t := append(binaryID, createHash(binaryID)...)
	return hex.EncodeToString(t), nil
}

func IsTokenValid(token string) bool {
	hash, err := hex.DecodeString(token)
	if err != nil {
		return false
	}
	return hmac.Equal(hash[binary.MaxVarintLen64:], createHash(hash[:binary.MaxVarintLen64]))
}

func GetIDFromToken(token string) (string, error) {
	hash, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(binary.BigEndian.Uint64(hash[:binary.MaxVarintLen64]), 10), nil
}

func createHash(v []byte) []byte {
	h := hmac.New(sha256.New, secretKey)
	h.Write(v)
	return h.Sum(nil)
}
