package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const bycrptCost = 12

var dummyHash = []byte("$2a$12$abcdede.4fYV4cZrTHuoDB.MvJzRn..01KYucHpKikuOEcCKKK")

func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bycrptCost)

	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func VerifyDummyPassword(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

func RandomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)

	if _, err := rand.Read(buf); err != nil {
		return "", nil
	}

	return hex.EncodeToString(buf), nil
}

func SHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
