package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

const minPasswordLength = 8

// var validate = newValidator()

type AppValidator struct {
	validate          *validator.Validate
	passwordCommonSet *PasswordCommonSet
}

func NewValidator(passwordCommonSet *PasswordCommonSet) *AppValidator {
	v := validator.New()

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]

		if name == "" || name == "-" {
			return field.Name
		}

		return name
	})

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
		return checkPasswordStrength(fl.Field().String(), *passwordCommonSet) == ""
	})

	// NIM --> tidak dimulai angka 0, tiga digit di tengah bukan 000, tiga digit di akhir bukan 000
	_ = v.RegisterValidation("nim", func(fl validator.FieldLevel) bool {
		return checkNIM(fl.Field().String())
	})

	return &AppValidator{
		validate:          v,
		passwordCommonSet: passwordCommonSet,
	}
}

func ValidateStruct(s any, appValidator AppValidator) map[string]string {
	validate := appValidator.validate

	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}

	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe, appValidator)
		}
	}

	return result
}

func messageFor(fe validator.FieldError, appValidator AppValidator) string {
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
			return checkPasswordStrength(value, *appValidator.passwordCommonSet)
		}
		return "password tidak memenuhi syarat"
	case "oneof":
		return "harus salah satu dari " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "len":
		if fe.Kind() == reflect.String {
			return "panjang harus tepat " + fe.Param() + " karakter"
		}
		return "panjang harus tepat " + fe.Param()
	case "numeric":
		return "karakter harus berupa angka"
	case "nim":
		return "pola tidak memenuhi"
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}

func checkPasswordStrength(password string, passwordCommonSet PasswordCommonSet) string {

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

	commonSet := passwordCommonSet.PasswordSet
	if _, exists := commonSet[password]; exists {
		return "password too common"
	}

	return ""
}

func checkNIM(nim string) bool {
	if nim[0] == '0' {
		return false
	}

	if nim[3:6] == "000" || nim[6:9] == "000" {
		return false
	}

	return true
}
