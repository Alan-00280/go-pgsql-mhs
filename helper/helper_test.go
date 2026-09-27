package helper

import (
	"path/filepath"
	"testing"
)

func TestCheckPasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{name: "valid password", password: "G00dP4ssWord", want: ""},
		{name: "too short", password: "Pass1", want: "minimum 8 character of password"},
		{name: "missing number", password: "Password", want: "passsword must contain mix of letter and numbers"},
		{name: "missing letter", password: "12345678", want: "passsword must contain mix of letter and numbers"},
		{name: "too weak", password: "qwerty123", want: "password too common"},
	}

	// Load Common Password
	passwordCommonPath, err := filepath.Abs("../files/common_password.txt")
	if err != nil {
		t.Errorf("can't load common password: %s", err)
	}

	passwordCommonSet, err := NewPasswordCommonSet(passwordCommonPath)
	if err != nil {
		t.Errorf("can't load common password: %s", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := checkPasswordStrength(test.password, *passwordCommonSet); got != test.want {
				t.Fatalf("checkPasswordStrength(%q) = %q, want %q", test.password, got, test.want)
			}
		})
	}
}
