package service

import (
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
)

type UserHandler struct {
	repo repository.UserRepository
}

func NewUserHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) ListAll() error {
	return nil
}

func (h *UserHandler) Get() error {
	return nil
}

// func (h *UserHandler) Create(c *fiber.Ctx, u model.User) model.User {
// 	ctx, cancel := helper.ReqCtx(c)
// 	defer cancel()

// 	new, err := h.repo.Create(ctx, u)
// 	if err != nil {
// 		return model.User{}
// 	}

// 	return new
// }

func (h *UserHandler) Replace() error {
	return nil
}

func (h *UserHandler) Update() error {
	return nil
}

func (h *UserHandler) List() error {
	return nil
}
