package model

import "time"

type Portofolio struct {
	ID           string        `json:"id"`
	UserID       int           `json:"user_id"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Project struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Technologies []string  `json:"technologies"`
	Link         string    `json:"link"`
	Year         int       `json:"year"`
	CreatedAt    time.Time `json:"created_at"`
}

type Certificate struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Publisher string    `json:"publisher"`
	Date      time.Time `json:"date"`
}

type CreatePortofolioReq struct {
	Headline     string                 `json:"headline" validate:"required,min=5,max=150"`
	Bio          string                 `json:"bio" validate:"required,min=5,max=500"`
	Certificates []CreateCertificateReq `json:"certificates" validate:"required"`
}

type CreateProjectReq struct {
	Title        string   `json:"title" validate:"required,min=5,max=150"`
	Technologies []string `json:"technologies" validate:"required,dive,min=3,max=150"`
	Link         string   `json:"link" validate:"required,min=5"`
	Year         int      `json:"year" validate:"required"`
}

type CreateCertificateReq struct {
	Title     string    `json:"title" validate:"required,min=5,max=150"`
	Publisher string    `json:"publisher" validate:"required,min=5,max=300"`
	Date      time.Time `json:"date" validate:"required"`
}

type AddProjectsReq struct {
	Project `json:"project" validate:"required"`
}

// Tidak Ada Patch
type UpdatePortofolioReq struct {
	Headline     string                 `json:"headline,omitempty" validate:"required,min=5,max=150"`
	Bio          string                 `json:"bio,omitempty" validate:"required,min=5,max=500"`
	Certificates []CreateCertificateReq `json:"certificates,omitempty" validate:"required,dive"`
}

// PortofolioQuery adalah parameter daftar portofolio seorang mahasiswa.
type PortofolioQuery struct {
	StudentID *int
	Before    string
	Limit     int
}

type PortofolioTechSummary struct {
	Count           int         `json:"count"`
	TopTechnologies []TechCount `json:"top_tech"`
}

type TechCount struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}
