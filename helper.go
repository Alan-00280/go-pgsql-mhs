package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/gofiber/fiber/v2"
)

func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func okList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func failValidation(c *fiber.Ctx, errors map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: true,
		Message: "validation fail",
		Errors:  errors,
	})
}

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: false,
		Message: message,
	})
}

func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

var allowedSort = map[string]bool{
	"id":    true,
	"nim":   true,
	"name":  true,
	"grade": true,
}

func parseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page"),
		Limit:  c.QueryInt("limit"),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}

	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}

	gradeFilter := model.GradeFilter{
		StartGrade: 0.00,
		EndGrade:   model.MAX_GRADE,
	}

	if gd_st := c.Query("grade_start"); gd_st != "" {
		if v, err := strconv.ParseFloat(gd_st, 64); err == nil {
			gradeFilter.StartGrade = v
		}
	}
	if gd_end := c.Query("grade_end"); gd_end != "" {
		if v, err := strconv.ParseFloat(gd_end, 64); err == nil {
			gradeFilter.EndGrade = v
		}
	}

	q.GradeFilter = &gradeFilter

	return q
}

func reqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
    return context.WithTimeout(c.UserContext(), 5*time.Second)
}

