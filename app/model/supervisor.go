package model

import "time"

type Supervisor struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Quota           int       `json:"quota"`
	SupervisedCount int       `json:"supervised_count"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

type StudentIDReq struct {
	StudentID int `json:"student_id"`
}

type AssignSupervisorReq struct {
	StudentID int `json:"student_id"`
}

type SwitchSupervisorReq struct {
	StudentID int `json:"student_id"`
	FromSupervisorID int `json:"from_supervisor_id"`
	ToSupervisorID int `json:"to_supervisor_id"`
}
