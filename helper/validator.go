package helper

import (
	"bufio"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

const minPasswordLength = 8

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]

		if name == "" || name == "-" {
			return field.Name
		}

		// self-made rules
		// nospace --> cek space / tab / new-line / return
		_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
			return !strings.Contains(fl.Field().String(), " \t\n\r")
		})

		// username --> cek hanya berupa letter / nomor / titik / underscore
		_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
			for _, r := range fl.Field().String() {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' {
					return false
				}
			}

			return true
		})

		// strongpassword --> menggunakan function checkPasswordStrength()
		_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
			return checkPasswordStrength(fl.Field().String()) != ""
		})

		// TODO
		// NIM
		// Tahun Angkatan

		return name
	})

	return v
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if fe.Kind() == reflect.String {
			return "minimal " + fe.Param() + " karakter"
		}
		return "minimal " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "maksimal " + fe.Param() + " karakter"
		}
		return "maksimal " + fe.Param()
	case "alphanum":
		return "wajib berisi huruf dan angka"
	case "nospace":
		return "tidak boleh berisi spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, garis bawah"
	case "strongpassword":
		if value, ok := fe.Value().(string); ok {
			return checkPasswordStrength(value)
		}
		return "password tidak memenuhi syarat"
	case "oneof":
		return "harus salah satu dari " + strings.ReplaceAll(fe.Param(), " ", ", ")
	// todo : NIM, Tahun Angkatan
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}

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

	password_path, err := filepath.Abs("./files/common_password.txt")
	if err != nil {
		panic(err)
	}

	file, err := os.Open(password_path)
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
