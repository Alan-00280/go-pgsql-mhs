package helper

import (
	"golang.org/x/crypto/bcrypt"
)

const bycrptCost = 12

var dummyHash = []byte("$2a$12$pRDImhzN94.4fYV4cZrTHuoDB.MvJzRn..01KYucHpKikuOEcCKKK")

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
