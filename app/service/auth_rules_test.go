package service

import (
	"testing"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
)

func TestValidateRegister(t *testing.T) {
	tests := []struct {
		name string
		req  model.RegisterReq
		want []string
	}{
		{
			name: "valid request",
			req: model.RegisterReq{
				Username: "john_doe",
				Email:    "john@example.com",
				Password: "G00dP4ssWord",
			},
		},
		{
			name: "username is required",
			req: model.RegisterReq{
				Email:    "john@example.com",
				Password: "G00dP4ssWord",
			},
			want: []string{"username"},
		},
		{
			name: "username too short",
			req: model.RegisterReq{
				Username: "jo",
				Email:    "john@example.com",
				Password: "G00dP4ssWord",
			},
			want: []string{"username"},
		},
		{
			name: "username contains invalid character",
			req: model.RegisterReq{
				Username: "john-doe",
				Email:    "john@example.com",
				Password: "G00dP4ssWord",
			},
			want: []string{"username"},
		},
		{
			name: "email is invalid",
			req: model.RegisterReq{
				Username: "john_doe",
				Email:    "not-an-email",
				Password: "G00dP4ssWord",
			},
			want: []string{"email"},
		},
		{
			name: "password is too short",
			req: model.RegisterReq{
				Username: "john_doe",
				Email:    "john@example.com",
				Password: "short",
			},
			want: []string{"password"},
		},
		{
			name: "password is too weak",
			req: model.RegisterReq{
				Username: "john_doe",
				Email:    "john@example.com",
				Password: "123456",
			},
			want: []string{"password"},
		},
		{
			name: "multiple errors are reported",
			req: model.RegisterReq{
				Username: "j",
				Email:    "invalid-email",
				Password: "short",
			},
			want: []string{"username", "email", "password"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertErrorKeys(t, helper.ValidateStruct(test.req), test.want)
		})
	}
}

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name string
		req  model.LoginReq
		want []string
	}{
		{
			name: "valid request",
			req: model.LoginReq{
				Username: "john_doe",
				Password: "G00dP4ssWord",
			},
		},
		{
			name: "username and password must be filled",
			req:  model.LoginReq{},
			want: []string{"username", "password"},
		},
		{
			name: "trims spaces before validation",
			req: model.LoginReq{
				Username: "   john_doe   ",
				Password: "   G00dP4ssWord   ",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertErrorKeys(t, helper.ValidateStruct(test.req), test.want)
		})
	}
}

// func TestCheckPasswordStrength(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		password string
// 		want     string
// 	}{
// 		{name: "valid password", password: "G00dP4ssWord", want: ""},
// 		{name: "too short", password: "Pass1", want: "minimum 8 character of password"},
// 		{name: "missing number", password: "Password", want: "passsword must contain mix of letter and numbers"},
// 		{name: "missing letter", password: "12345678", want: "passsword must contain mix of letter and numbers"},
// 		{name: "too weak", password: "qwerty123", want: "password too common"},
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			if got := checkPasswordStrength(test.password); got != test.want {
// 				t.Fatalf("checkPasswordStrength(%q) = %q, want %q", test.password, got, test.want)
// 			}
// 		})
// 	}
// }
