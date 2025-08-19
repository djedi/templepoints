package main

import (
	"time"
)

type Ward struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Points       int       `json:"points"`
	PendingPoints int      `json:"pending_points"`
	CreatedAt    time.Time `json:"created_at"`
}

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"` // "admin" or "ward_approver"
	WardID    *int      `json:"ward_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PointSubmission struct {
	ID           int       `json:"id"`
	WardID       int       `json:"ward_id"`
	WardName     string    `json:"ward_name,omitempty"`
	SubmitterName string   `json:"submitter_name"`
	Points       int       `json:"points"`
	Note         string    `json:"note"`
	Status       string    `json:"status"` // "pending", "approved", "rejected"
	ApprovedBy   *int      `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type LeaderboardEntry struct {
	Rank          int       `json:"rank"`
	WardID        int       `json:"ward_id"`
	WardName      string    `json:"ward_name"`
	Points        int       `json:"points"`
	PendingPoints int       `json:"pending_points"`
	TotalPoints   int       `json:"total_points"`
	Progress      float64   `json:"progress"` // percentage to 1300
	Achievements  []string  `json:"achievements"`
	Streak        int       `json:"streak"`
	LastActivity  time.Time `json:"last_activity"`
}

type Stats struct {
	LeadingWard   string `json:"leading_ward"`
	TotalPoints   int    `json:"total_points"`
	DaysActive    int    `json:"days_active"`
	Participants  int    `json:"participants"`
}

type Achievement struct {
	ID          int       `json:"id"`
	WardID      int       `json:"ward_id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	EarnedAt    time.Time `json:"earned_at"`
}

type ActivityLog struct {
	ID        int       `json:"id"`
	WardID    int       `json:"ward_id"`
	UserID    *int      `json:"user_id,omitempty"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}