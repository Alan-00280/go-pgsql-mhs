package service

import (
	"bufio"
	"net/mail"
	"os"
	"strings"
	"unicode"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
)

// TODO
// buat konstanta minPasswordLength bernilai 8
const minPasswordLength = 8

// buat functions yang mengembalikan errs map[string]string
// antara lain:
//  1. ValidateRegister(model.RegisterRequest):
//     trim username
//     username wajib diisi, username min 3 karakter, username terdiri huruf angka titik underscore (definisikan helper func di bawah)
//     email format validity
//     password strength (helper func)
func ValidateRegister(req model.RegisterReq) map[string]string {
	errs := map[string]string{}

	req.Username = strings.TrimSpace(req.Username)
	switch {
	case req.Username == "":
		errs["username"] = "username wajib diisi"
	case len(req.Username) < 3:
		errs["username"] = "username wajib memiliki panjang minimal 3 karakter"
	case !isValidUsername(req.Username):
		errs["username"] = "username hanya bisa mengandung huruf, angka, titik, dan underscore"
	}

	if !isValidEmail(req.Email) {
		errs["email"] = "email invalid!"
	}

	if msg := checkPasswordStrength(req.Password); msg != "" {
		errs["password"] = msg
	}

	return errs
}

//  2. ValidateLogin(model.LoginRequest)
//     cek kelengkapan
func ValidateLogin(req model.LoginReq) map[string]string {
	errs := map[string]string{}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		errs["username"] = "username belum diisi"
	}

	req.Password = strings.TrimSpace(req.Password)
	if req.Password == "" {
		errs["password"] = "password belum diisi"
	}

	return errs
}

// helper:
// checkPasswordStrength(password string) string
// isValidUsername(username string) bool
// isValidEmail(email string) bool

func checkPasswordStrength(password string) string {
	if len(password) < minPasswordLength {
		return "minimum 8 character of password"
	}

	var hasLetter, hasDigit bool = false, false
	for _, c := range password {
		switch {
		case unicode.IsLetter(c):
			hasLetter = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return "passsword must contain mix of letter and numbers"
	}

	file, err := os.Open("../../files/common_password.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		weak_password := scanner.Text()
		if password == weak_password {
			return "password too common"
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return ""
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isValidUsername(username string) bool {

	for _, c := range username {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '.' && c != '_' {
			return false
		}
	}

	return true
}
