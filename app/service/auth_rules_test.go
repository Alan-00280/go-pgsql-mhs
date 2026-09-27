package service

import (
	"path/filepath"
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

	// Load Common Password
	passwordCommonPath, err := filepath.Abs("../../files/common_password.txt")
	if err != nil {
		t.Errorf("can't load common password: %s", err)
	}

	passwordCommonSet, err := helper.NewPasswordCommonSet(passwordCommonPath)
	if err != nil {
		t.Errorf("can't load common password: %s", err)
	}

	// App Validator
	appValidator := helper.NewValidator(passwordCommonSet)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertErrorKeys(t, helper.ValidateStruct(test.req, *appValidator), test.want)
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

	// Load Common Password
	passwordCommonPath, err := filepath.Abs("../../files/common_password.txt")
	if err != nil {
		t.Errorf("can't load common password: %s", err)
	}

	passwordCommonSet, err := helper.NewPasswordCommonSet(passwordCommonPath)
	if err != nil {
		t.Errorf("can't load common password: %s", err)
	}

	// App Validator
	appValidator := helper.NewValidator(passwordCommonSet)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertErrorKeys(t, helper.ValidateStruct(test.req, *appValidator), test.want)
		})
	}
}
