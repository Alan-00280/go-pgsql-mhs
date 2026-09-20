package model

import "time"

const MAX_GRADE = 4.00
const NIM_LENGTH = 9

type Student struct {
	ID        int        `json:"id"`
	NIM       string     `json:"nim"`
	Name      string     `json:"name"`
	Grade     float64    `json:"grade"`
	IsActive  bool       `json:"is_active"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	OwnerID   int        `json:"owner_id"`
}

// POST - For Creating Student
type CreateStudentReq struct {
	NIM   string  `json:"nim"`
	Name  string  `json:"name"`
	Grade float64 `json:"grade"`
}

// PUT - For Update All Data of Student
type ReplaceStudentReq struct {
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

// PATCH - For Update Some Data of Student
type PatchStudentReq struct {
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type StudentFilter struct {
	StartGrade float64
	EndGrade   float64
	OwnerID    *int
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}
